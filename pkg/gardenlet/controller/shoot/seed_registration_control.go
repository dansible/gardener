// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shoot

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	v1alpha1constants "github.com/gardener/gardener/pkg/apis/core/v1alpha1/constants"
	gardencorev1alpha1helper "github.com/gardener/gardener/pkg/apis/core/v1alpha1/helper"
	"github.com/gardener/gardener/pkg/chartrenderer"
	gardencoreinformers "github.com/gardener/gardener/pkg/client/core/informers/externalversions/core/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/gardenlet/apis/config"
	configv1alpha1 "github.com/gardener/gardener/pkg/gardenlet/apis/config/v1alpha1"
	"github.com/gardener/gardener/pkg/gardenlet/bootstrap"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/version"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	bootstraptokenapi "k8s.io/cluster-bootstrap/token/api"
	bootstraptokenutil "k8s.io/cluster-bootstrap/token/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (c *Controller) seedRegistrationAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	namespace, _, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return
	}
	if namespace != v1alpha1constants.GardenNamespace {
		return
	}

	c.seedRegistrationQueue.Add(key)
}

func (c *Controller) seedRegistrationUpdate(oldObj, newObj interface{}) {
	oldShoot, ok := oldObj.(*gardencorev1alpha1.Shoot)
	if !ok {
		return
	}
	newShoot, ok := newObj.(*gardencorev1alpha1.Shoot)
	if !ok {
		return
	}

	if newShoot.Generation == newShoot.Status.ObservedGeneration && apiequality.Semantic.DeepEqual(newShoot.Annotations, oldShoot.Annotations) {
		return
	}

	c.seedRegistrationAdd(newObj)
}

func (c *Controller) reconcileShootedSeedRegistrationKey(req reconcile.Request) (reconcile.Result, error) {
	shoot, err := c.shootLister.Shoots(req.Namespace).Get(req.Name)
	if apierrors.IsNotFound(err) {
		logger.Logger.Debugf("[SHOOTED SEED REGISTRATION] %s/%s - skipping because Shoot has been deleted", req.Namespace, req.Name)
		return reconcile.Result{}, nil
	}
	if err != nil {
		logger.Logger.Errorf("[SHOOTED SEED REGISTRATION] %s/%s - unable to retrieve object from store: %v", req.Namespace, req.Name, err)
		return reconcile.Result{}, err
	}

	shootedSeedConfig, err := gardencorev1alpha1helper.ReadShootedSeed(shoot)
	if err != nil {
		return reconcile.Result{}, err
	}

	return c.seedRegistrationControl.Reconcile(shoot, shootedSeedConfig)
}

// SeedRegistrationControlInterface implements the control logic for requeuing shooted Seeds after extensions have been updated.
// It is implemented as an interface to allow for extensions that provide different semantics. Currently, there is only one
// implementation.
type SeedRegistrationControlInterface interface {
	Reconcile(shootObj *gardencorev1alpha1.Shoot, shootedSeedConfig *gardencorev1alpha1helper.ShootedSeed) (reconcile.Result, error)
}

// NewDefaultSeedRegistrationControl returns a new instance of the default implementation SeedRegistrationControlInterface that
// implements the documented semantics for registering shooted seeds. You should use an instance returned from
// NewDefaultSeedRegistrationControl() for any scenario other than testing.
func NewDefaultSeedRegistrationControl(k8sGardenClient kubernetes.Interface, k8sGardenCoreInformers gardencoreinformers.Interface, imageVector imagevector.ImageVector, config *config.GardenletConfiguration, recorder record.EventRecorder) SeedRegistrationControlInterface {
	return &defaultSeedRegistrationControl{k8sGardenClient, k8sGardenCoreInformers, imageVector, config, recorder}
}

type defaultSeedRegistrationControl struct {
	k8sGardenClient        kubernetes.Interface
	k8sGardenCoreInformers gardencoreinformers.Interface
	imageVector            imagevector.ImageVector
	config                 *config.GardenletConfiguration
	recorder               record.EventRecorder
}

func (c *defaultSeedRegistrationControl) Reconcile(shootObj *gardencorev1alpha1.Shoot, shootedSeedConfig *gardencorev1alpha1helper.ShootedSeed) (reconcile.Result, error) {
	var (
		ctx         = context.TODO()
		shoot       = shootObj.DeepCopy()
		shootLogger = logger.NewShootLogger(logger.Logger, shoot.Name, shoot.Namespace)
	)

	if shoot.DeletionTimestamp == nil && shootedSeedConfig != nil {
		if shoot.Status.LastOperation == nil || shoot.Status.LastOperation.State != gardencorev1alpha1.LastOperationStateSucceeded {
			shootLogger.Infof("[SHOOTED SEED REGISTRATION] Waiting for shoot %s to be reconciled before registering it as seed", shoot.Name)
			return reconcile.Result{
				RequeueAfter: 10 * time.Second,
			}, nil
		}

		if shootedSeedConfig.NoGardenlet {
			shootLogger.Infof("[SHOOTED SEED REGISTRATION] Registering %s as seed as configuration says that no gardenlet is desired", shoot.Name)
			if err := registerAsSeed(ctx, c.k8sGardenClient, shoot, shootedSeedConfig); err != nil {
				message := fmt.Sprintf("Could not register shoot %q as seed: %+v", shoot.Name, err)
				shootLogger.Errorf(message)
				c.recorder.Event(shoot, corev1.EventTypeWarning, "SeedRegistration", message)
				return reconcile.Result{}, err
			}
		} else {
			shootLogger.Infof("[SHOOTED SEED REGISTRATION] Deploying gardenlet into %s which will register shoot as seed", shoot.Name)
			if err := deployGardenlet(ctx, c.k8sGardenClient, shoot, shootedSeedConfig, c.imageVector, c.config); err != nil {
				message := fmt.Sprintf("Could not deploy Gardenlet into shoot %q: %+v", shoot.Name, err)
				shootLogger.Errorf(message)
				c.recorder.Event(shoot, corev1.EventTypeWarning, "GardenletDeployment", message)
				return reconcile.Result{}, err
			}
		}
	} else {
		shootLogger.Infof("[SHOOTED SEED REGISTRATION] Deleting `Seed` object for %s", shoot.Name)
		if err := deregisterAsSeed(ctx, c.k8sGardenClient, shoot); err != nil {
			message := fmt.Sprintf("Could not deregister shoot %q as seed: %+v", shoot.Name, err)
			shootLogger.Errorf(message)
			c.recorder.Event(shoot, corev1.EventTypeWarning, "SeedDeletion", message)
			return reconcile.Result{}, err
		}

		if err := checkSeedAssociations(ctx, c.k8sGardenClient, shoot.Name); err != nil {
			message := fmt.Sprintf("Error during check for associated resources for the to-be-deleted shooted seed %q: %+v", shoot.Name, err)
			shootLogger.Errorf(message)
			c.recorder.Event(shoot, corev1.EventTypeWarning, "SeedDeletion", message)
			return reconcile.Result{}, err
		}

		shootLogger.Infof("[SHOOTED SEED REGISTRATION] Deleting gardenlet in seed %s", shoot.Name)
		if err := deleteGardenlet(ctx, c.k8sGardenClient, shoot); err != nil {
			message := fmt.Sprintf("Could not deregister shoot %q as seed: %+v", shoot.Name, err)
			shootLogger.Errorf(message)
			c.recorder.Event(shoot, corev1.EventTypeWarning, "GardenletDeletion", message)
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func getShootSecret(ctx context.Context, k8sGardenClient kubernetes.Interface, shoot *gardencorev1alpha1.Shoot) (*corev1.Secret, error) {
	shootSecretBinding := &gardencorev1alpha1.SecretBinding{}
	if err := k8sGardenClient.Client().Get(ctx, kutil.Key(shoot.Namespace, shoot.Spec.SecretBindingName), shootSecretBinding); err != nil {
		return nil, err
	}
	shootSecret := &corev1.Secret{}
	err := k8sGardenClient.Client().Get(ctx, kutil.Key(shootSecretBinding.SecretRef.Namespace, shootSecretBinding.SecretRef.Name), shootSecret)
	return shootSecret, err
}

func applySeedBackupConfig(ctx context.Context, k8sGardenClient kubernetes.Interface, shoot *gardencorev1alpha1.Shoot, shootSecret *corev1.Secret, shootedSeedConfig *gardencorev1alpha1helper.ShootedSeed) (*gardencorev1alpha1.SeedBackup, error) {
	var backupProfile *gardencorev1alpha1.SeedBackup
	if shootedSeedConfig.Backup != nil {
		backupProfile = shootedSeedConfig.Backup.DeepCopy()

		if len(backupProfile.Provider) == 0 {
			backupProfile.Provider = shoot.Spec.Provider.Type
		}

		if len(backupProfile.SecretRef.Name) == 0 || len(backupProfile.SecretRef.Namespace) == 0 {
			var (
				backupSecretName      = fmt.Sprintf("backup-%s", shoot.Name)
				backupSecretNamespace = v1alpha1constants.GardenNamespace
			)

			backupSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      backupSecretName,
					Namespace: backupSecretNamespace,
				},
			}

			if _, err := controllerutil.CreateOrUpdate(ctx, k8sGardenClient.Client(), backupSecret, func() error {
				backupSecret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
					*metav1.NewControllerRef(shoot, gardencorev1alpha1.SchemeGroupVersion.WithKind("Shoot")),
				}
				backupSecret.Type = corev1.SecretTypeOpaque
				backupSecret.Data = shootSecret.Data
				return nil
			}); err != nil {
				return nil, err
			}

			backupProfile.SecretRef.Name = backupSecretName
			backupProfile.SecretRef.Namespace = backupSecretNamespace
		}
	}

	return backupProfile, nil
}

func applySeedSecret(ctx context.Context, k8sGardenClient kubernetes.Interface, shoot *gardencorev1alpha1.Shoot, shootSecret *corev1.Secret, secretName, secretNamespace string) error {
	shootKubeconfigSecret := &corev1.Secret{}
	if err := k8sGardenClient.Client().Get(ctx, kutil.Key(shoot.Namespace, fmt.Sprintf("%s.kubeconfig", shoot.Name)), shootKubeconfigSecret); err != nil {
		return err
	}

	seedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
	}

	return kutil.CreateOrUpdate(ctx, k8sGardenClient.Client(), seedSecret, func() error {
		seedSecret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(shoot, gardencorev1alpha1.SchemeGroupVersion.WithKind("Shoot")),
		}
		seedSecret.Type = corev1.SecretTypeOpaque
		seedSecret.Data = shootSecret.Data
		seedSecret.Data[kubernetes.KubeConfig] = shootKubeconfigSecret.Data[kubernetes.KubeConfig]
		return nil
	})
}

func prepareSeedConfig(ctx context.Context, k8sGardenClient kubernetes.Interface, shoot *gardencorev1alpha1.Shoot, shootedSeedConfig *gardencorev1alpha1helper.ShootedSeed, secretRef *corev1.SecretReference) (*gardencorev1alpha1.SeedSpec, error) {
	shootSecret, err := getShootSecret(ctx, k8sGardenClient, shoot)
	if err != nil {
		return nil, err
	}

	backupProfile, err := applySeedBackupConfig(ctx, k8sGardenClient, shoot, shootSecret, shootedSeedConfig)
	if err != nil {
		return nil, err
	}

	if secretRef != nil {
		if err := applySeedSecret(ctx, k8sGardenClient, shoot, shootSecret, secretRef.Name, secretRef.Namespace); err != nil {
			return nil, err
		}
	}

	var taints []gardencorev1alpha1.SeedTaint
	if shootedSeedConfig.DisableDNS != nil && *shootedSeedConfig.DisableDNS {
		taints = append(taints, gardencorev1alpha1.SeedTaint{Key: gardencorev1alpha1.SeedTaintDisableDNS})
	}
	if shootedSeedConfig.Protected != nil && *shootedSeedConfig.Protected {
		taints = append(taints, gardencorev1alpha1.SeedTaint{Key: gardencorev1alpha1.SeedTaintProtected})
	}
	if shootedSeedConfig.Visible != nil && !*shootedSeedConfig.Visible {
		taints = append(taints, gardencorev1alpha1.SeedTaint{Key: gardencorev1alpha1.SeedTaintInvisible})
	}

	var volume *gardencorev1alpha1.SeedVolume
	if shootedSeedConfig.MinimumVolumeSize != nil {
		minimumSize, err := resource.ParseQuantity(*shootedSeedConfig.MinimumVolumeSize)
		if err != nil {
			return nil, err
		}
		volume = &gardencorev1alpha1.SeedVolume{
			MinimumSize: &minimumSize,
		}
	}

	return &gardencorev1alpha1.SeedSpec{
		Provider: gardencorev1alpha1.SeedProvider{
			Type:   shoot.Spec.Provider.Type,
			Region: shoot.Spec.Region,
		},
		DNS: gardencorev1alpha1.SeedDNS{
			IngressDomain: fmt.Sprintf("%s.%s", common.IngressPrefix, *(shoot.Spec.DNS.Domain)),
		},
		SecretRef: secretRef,
		Networks: gardencorev1alpha1.SeedNetworks{
			Pods:          *shoot.Spec.Networking.Pods,
			Services:      *shoot.Spec.Networking.Services,
			Nodes:         shoot.Spec.Networking.Nodes,
			ShootDefaults: shootedSeedConfig.ShootDefaults,
		},
		BlockCIDRs: shootedSeedConfig.BlockCIDRs,
		Taints:     taints,
		Backup:     backupProfile,
		Volume:     volume,
	}, nil
}

// registerAsSeed registers a Shoot cluster as a Seed in the Garden cluster.
func registerAsSeed(ctx context.Context, k8sGardenClient kubernetes.Interface, shoot *gardencorev1alpha1.Shoot, shootedSeedConfig *gardencorev1alpha1helper.ShootedSeed) error {
	if shoot.Spec.DNS == nil || shoot.Spec.DNS.Domain == nil {
		return errors.New("cannot register Shoot as Seed if it does not specify a domain")
	}

	var (
		secretRef = &corev1.SecretReference{
			Name:      fmt.Sprintf("seed-%s", shoot.Name),
			Namespace: v1alpha1constants.GardenNamespace,
		}

		seed = &gardencorev1alpha1.Seed{
			ObjectMeta: metav1.ObjectMeta{
				Name: shoot.Name,
			},
		}
	)

	seedSpec, err := prepareSeedConfig(ctx, k8sGardenClient, shoot, shootedSeedConfig, secretRef)
	if err != nil {
		return err
	}

	_, err = controllerutil.CreateOrUpdate(ctx, k8sGardenClient.Client(), seed, func() error {
		seed.Labels = utils.MergeStringMaps(shoot.Labels, map[string]string{
			v1alpha1constants.DeprecatedGardenRole: v1alpha1constants.GardenRoleSeed,
			v1alpha1constants.GardenRole:           v1alpha1constants.GardenRoleSeed,
		})
		seed.Spec = *seedSpec
		return nil
	})
	return err
}

// deregisterAsSeed de-registers a Shoot cluster as a Seed in the Garden cluster.
func deregisterAsSeed(ctx context.Context, k8sGardenClient kubernetes.Interface, shoot *gardencorev1alpha1.Shoot) error {
	seed, err := k8sGardenClient.GardenCore().CoreV1alpha1().Seeds().Get(shoot.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if err := k8sGardenClient.GardenCore().CoreV1alpha1().Seeds().Delete(seed.Name, nil); client.IgnoreNotFound(err) != nil {
		return err
	}

	var secretRefs []corev1.SecretReference
	if seed.Spec.SecretRef != nil {
		secretRefs = append(secretRefs, *seed.Spec.SecretRef)
	}
	if seed.Spec.Backup != nil {
		secretRefs = append(secretRefs, seed.Spec.Backup.SecretRef)
	}

	for _, secretRef := range secretRefs {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretRef.Name,
				Namespace: secretRef.Namespace,
			},
		}
		if err := k8sGardenClient.Client().Delete(ctx, secret, kubernetes.DefaultDeleteOptions...); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	return nil
}

const (
	gardenletKubeconfigBootstrapSecretName = "gardenlet-kubeconfig-bootstrap"
	gardenletKubeconfigSecretName          = "gardenlet-kubeconfig"
)

func deployGardenlet(ctx context.Context, k8sGardenClient kubernetes.Interface, shoot *gardencorev1alpha1.Shoot, shootedSeedConfig *gardencorev1alpha1helper.ShootedSeed, imageVector imagevector.ImageVector, cfg *config.GardenletConfiguration) error {
	k8sSeedClient, err := kubernetes.NewClientFromSecret(k8sGardenClient, shoot.Namespace, fmt.Sprintf("%s.kubeconfig", shoot.Name),
		kubernetes.WithClientOptions(client.Options{
			Scheme: kubernetes.SeedScheme,
		}),
	)
	if err != nil {
		return err
	}

	// create bootstrap token and bootstrap kubeconfig in case there is no existing gardenlet kubeconfig yet
	var bootstrapKubeconfigValues map[string]interface{}
	if err := k8sSeedClient.Client().Get(ctx, kutil.Key(v1alpha1constants.GardenNamespace, gardenletKubeconfigSecretName), &corev1.Secret{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		var (
			tokenID               = utils.ComputeSHA256Hex([]byte(shoot.Name))[:6]
			validity              = 24 * time.Hour
			refreshBootstrapToken = true
			bootstrapTokenSecret  *corev1.Secret
		)

		secret := &corev1.Secret{}
		if err := k8sGardenClient.Client().Get(ctx, kutil.Key(metav1.NamespaceSystem, bootstraptokenutil.BootstrapTokenSecretName(tokenID)), secret); client.IgnoreNotFound(err) != nil {
			return err
		}

		if expirationTime, ok := secret.Data[bootstraptokenapi.BootstrapTokenExpirationKey]; ok {
			t, err := time.Parse(time.RFC3339, string(expirationTime))
			if err != nil {
				return err
			}

			if !t.Before(metav1.Now().UTC()) {
				bootstrapTokenSecret = secret
				refreshBootstrapToken = false
			}
		}

		if refreshBootstrapToken {
			bootstrapTokenSecret, err = kutil.ComputeBootstrapToken(ctx, k8sGardenClient.Client(), tokenID, fmt.Sprintf("A bootstrap token for the Gardenlet for shooted seed %q.", shoot.Name), validity)
			if err != nil {
				return err
			}
		}

		restConfig := *k8sGardenClient.RESTConfig()
		if addr := cfg.GardenClientConnection.GardenClusterAddress; addr != nil {
			restConfig.Host = *addr
		}
		if caCert := cfg.GardenClientConnection.GardenClusterCACert; caCert != nil {
			restConfig.TLSClientConfig = rest.TLSClientConfig{
				CAData: caCert,
			}
		}

		bootstrapKubeconfig, err := bootstrap.MarshalKubeconfigFromBootstrapToken(&restConfig, kutil.BootstrapTokenFrom(bootstrapTokenSecret.Data))
		if err != nil {
			return err
		}

		bootstrapKubeconfigValues = map[string]interface{}{
			"name":       gardenletKubeconfigBootstrapSecretName,
			"namespace":  v1alpha1constants.GardenNamespace,
			"kubeconfig": string(bootstrapKubeconfig),
		}
	}

	renderer, err := chartrenderer.NewForConfig(k8sSeedClient.RESTConfig())
	if err != nil {
		return err
	}
	applier, err := kubernetes.NewApplierForConfig(k8sSeedClient.RESTConfig())
	if err != nil {
		return err
	}
	chartApplier := kubernetes.NewChartApplier(renderer, applier)

	// convert config from internal version to v1alpha1 as Helm chart is based on v1alpha1
	scheme := runtime.NewScheme()
	if err := config.AddToScheme(scheme); err != nil {
		return err
	}
	if err := configv1alpha1.AddToScheme(scheme); err != nil {
		return err
	}
	external, err := scheme.ConvertToVersion(cfg, configv1alpha1.SchemeGroupVersion)
	if err != nil {
		return err
	}
	externalConfig, ok := external.(*configv1alpha1.GardenletConfiguration)
	if !ok {
		return fmt.Errorf("error converting config to external version")
	}

	gardenletImage, err := imageVector.FindImage("gardenlet")
	if err != nil {
		return err
	}

	var secretRef *corev1.SecretReference
	if shootedSeedConfig.WithSecretRef {
		secretRef = &corev1.SecretReference{
			Name:      fmt.Sprintf("seed-%s", shoot.Name),
			Namespace: v1alpha1constants.GardenNamespace,
		}
	}

	seedSpec, err := prepareSeedConfig(ctx, k8sGardenClient, shoot, shootedSeedConfig, secretRef)
	if err != nil {
		return err
	}

	var imageVectorOverwrite string
	if overWritePath := os.Getenv(imagevector.OverrideEnv); len(overWritePath) > 0 {
		data, err := ioutil.ReadFile(overWritePath)
		if err != nil {
			return err
		}
		imageVectorOverwrite = string(data)
	}

	values := map[string]interface{}{
		"global": map[string]interface{}{
			"gardenlet": map[string]interface{}{
				"image": map[string]interface{}{
					"repository": gardenletImage.String(),
					"tag":        version.Get().GitVersion,
				},
				"revisionHistoryLimit": 0,
				"vpa":                  true,
				"imageVectorOverwrite": imageVectorOverwrite,
				"config": map[string]interface{}{
					"gardenClientConnection": map[string]interface{}{
						"acceptContentTypes":   externalConfig.GardenClientConnection.AcceptContentTypes,
						"contentType":          externalConfig.GardenClientConnection.ContentType,
						"qps":                  externalConfig.GardenClientConnection.QPS,
						"burst":                externalConfig.GardenClientConnection.Burst,
						"gardenClusterAddress": externalConfig.GardenClientConnection.GardenClusterAddress,
						"bootstrapKubeconfig":  bootstrapKubeconfigValues,
						"kubeconfigSecret": map[string]interface{}{
							"name":      gardenletKubeconfigSecretName,
							"namespace": v1alpha1constants.GardenNamespace,
						},
					},
					"seedClientConnection":  externalConfig.SeedClientConnection.ClientConnectionConfiguration,
					"shootClientConnection": externalConfig.ShootClientConnection,
					"controllers":           externalConfig.Controllers,
					"leaderElection":        externalConfig.LeaderElection,
					"discovery":             externalConfig.Discovery,
					"logLevel":              externalConfig.LogLevel,
					"kubernetesLogLevel":    externalConfig.KubernetesLogLevel,
					"featureGates":          externalConfig.FeatureGates,
					"seedConfig": &configv1alpha1.SeedConfig{
						Seed: gardencorev1alpha1.Seed{
							ObjectMeta: metav1.ObjectMeta{
								Name:   shoot.Name,
								Labels: shoot.Labels,
							},
							Spec: *seedSpec,
						},
					},
				},
			},
		},
	}

	gardenNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: v1alpha1constants.GardenNamespace}}
	if err := k8sSeedClient.Client().Create(ctx, gardenNamespace); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return chartApplier.ApplyChart(ctx, filepath.Join(common.ChartPath, "gardener", "gardenlet"), v1alpha1constants.GardenNamespace, "gardenlet", values, nil)
}

func deleteGardenlet(ctx context.Context, k8sGardenClient kubernetes.Interface, shoot *gardencorev1alpha1.Shoot) error {
	k8sSeedClient, err := kubernetes.NewClientFromSecret(k8sGardenClient, shoot.Namespace, fmt.Sprintf("%s.kubeconfig", shoot.Name),
		kubernetes.WithClientOptions(client.Options{
			Scheme: kubernetes.SeedScheme,
		}),
	)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	vpa := &unstructured.Unstructured{}
	vpa.SetAPIVersion("autoscaling.k8s.io/v1beta2")
	vpa.SetKind("VerticalPodAutoscaler")
	vpa.SetName("gardenlet-vpa")
	vpa.SetNamespace(v1alpha1constants.GardenNamespace)

	for _, obj := range []runtime.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "gardenlet", Namespace: v1alpha1constants.GardenNamespace}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "gardenlet-configmap", Namespace: v1alpha1constants.GardenNamespace}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "gardenlet-imagevector-overwrite", Namespace: v1alpha1constants.GardenNamespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: gardenletKubeconfigBootstrapSecretName, Namespace: v1alpha1constants.GardenNamespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: gardenletKubeconfigSecretName, Namespace: v1alpha1constants.GardenNamespace}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "gardenlet", Namespace: v1alpha1constants.GardenNamespace}},
		&policyv1beta1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Name: "gardenlet", Namespace: v1alpha1constants.GardenNamespace}},
		vpa,
	} {
		if err := k8sSeedClient.Client().Delete(ctx, obj); client.IgnoreNotFound(err) != nil && !meta.IsNoMatchError(err) {
			return err
		}
	}

	return nil
}

func checkSeedAssociations(ctx context.Context, k8sGardenClient kubernetes.Interface, seedName string) error {
	var (
		results []string
		err     error
	)

	for name, f := range map[string]func(context.Context, client.Client, string) ([]string, error){
		"BackupBuckets":           controllerutils.DetermineBackupBucketAssociations,
		"BackupEntries":           controllerutils.DetermineBackupEntryAssociations,
		"ControllerInstallations": controllerutils.DetermineControllerInstallationAssociations,
		"Shoots":                  controllerutils.DetermineShootAssociations,
	} {
		results, err = f(ctx, k8sGardenClient.Client(), seedName)
		if err != nil {
			return err
		}

		if len(results) > 0 {
			return fmt.Errorf("Still associated %s with seed %q: %+v", name, seedName, results)
		}
	}

	return nil
}
