// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gardener/gardener/cmd/utils"
	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardencoreinformers "github.com/gardener/gardener/pkg/client/core/informers/externalversions"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/controllermanager/apis/config"
	controllermanagerconfigv1alpha1 "github.com/gardener/gardener/pkg/controllermanager/apis/config/v1alpha1"
	"github.com/gardener/gardener/pkg/controllermanager/controller"
	"github.com/gardener/gardener/pkg/controllermanager/features"
	"github.com/gardener/gardener/pkg/controllermanager/server/handlers/webhooks"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/pkg/server"
	"github.com/gardener/gardener/pkg/server/handlers"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery"
	diskcache "k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/informers"
	kubeinformers "k8s.io/client-go/informers"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Options has all the context and parameters needed to run a Gardener controller manager.
type Options struct {
	// ConfigFile is the location of the Gardener controller manager's configuration file.
	ConfigFile string
	config     *config.ControllerManagerConfiguration
	scheme     *runtime.Scheme
	codecs     serializer.CodecFactory
}

// AddFlags adds flags for a specific Gardener controller manager to the specified FlagSet.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ConfigFile, "config", o.ConfigFile, "The path to the configuration file.")
}

// NewOptions returns a new Options object.
func NewOptions() (*Options, error) {
	o := &Options{
		config: new(config.ControllerManagerConfiguration),
	}

	o.scheme = runtime.NewScheme()
	o.codecs = serializer.NewCodecFactory(o.scheme)

	if err := config.AddToScheme(o.scheme); err != nil {
		return nil, err
	}
	if err := controllermanagerconfigv1alpha1.AddToScheme(o.scheme); err != nil {
		return nil, err
	}
	if err := gardencorev1alpha1.AddToScheme(scheme.Scheme); err != nil {
		return nil, err
	}

	return o, nil
}

// loadConfigFromFile loads the contents of file and decodes it as a
// ControllerManagerConfiguration object.
func (o *Options) loadConfigFromFile(file string) (*config.ControllerManagerConfiguration, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return o.decodeConfig(data)
}

// decodeConfig decodes data as a ControllerManagerConfiguration object.
func (o *Options) decodeConfig(data []byte) (*config.ControllerManagerConfiguration, error) {
	configObj, gvk, err := o.codecs.UniversalDecoder().Decode(data, nil, nil)
	if err != nil {
		return nil, err
	}
	config, ok := configObj.(*config.ControllerManagerConfiguration)
	if !ok {
		return nil, fmt.Errorf("got unexpected config type: %v", gvk)
	}
	return config, nil
}

func (o *Options) configFileSpecified() error {
	if len(o.ConfigFile) == 0 {
		return fmt.Errorf("missing Gardener controller manager config file")
	}
	return nil
}

// Validate validates all the required options.
func (o *Options) validate(args []string) error {
	if len(args) != 0 {
		return errors.New("arguments are not supported")
	}

	return nil
}

func (o *Options) applyDefaults(in *config.ControllerManagerConfiguration) (*config.ControllerManagerConfiguration, error) {
	external, err := o.scheme.ConvertToVersion(in, controllermanagerconfigv1alpha1.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	o.scheme.Default(external)

	internal, err := o.scheme.ConvertToVersion(external, config.SchemeGroupVersion)
	if err != nil {
		return nil, err
	}
	out := internal.(*config.ControllerManagerConfiguration)

	return out, nil
}

func (o *Options) run(ctx context.Context, cancel context.CancelFunc) error {
	if len(o.ConfigFile) > 0 {
		c, err := o.loadConfigFromFile(o.ConfigFile)
		if err != nil {
			return err
		}
		o.config = c
	}

	// Add feature flags
	if err := features.FeatureGate.SetFromMap(o.config.FeatureGates); err != nil {
		return err
	}

	gardener, err := NewGardener(o.config)
	if err != nil {
		return err
	}

	return gardener.Run(ctx, cancel)
}

// NewCommandStartGardenerControllerManager creates a *cobra.Command object with default parameters
func NewCommandStartGardenerControllerManager(ctx context.Context, cancel context.CancelFunc) *cobra.Command {
	opts, err := NewOptions()
	if err != nil {
		panic(err)
	}

	cmd := &cobra.Command{
		Use:   "gardener-controller-manager",
		Short: "Launch the Gardener controller manager",
		Long: `In essence, the Gardener is an extension API server along with a bundle
of Kubernetes controllers which introduce new API objects in an existing Kubernetes
cluster (which is called Garden cluster) in order to use them for the management of
further Kubernetes clusters (which are called Shoot clusters).
To do that reliably and to offer a certain quality of service, it requires to control
the main components of a Kubernetes cluster (etcd, API server, controller manager, scheduler).
These so-called control plane components are hosted in Kubernetes clusters themselves
(which are called Seed clusters).`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.configFileSpecified(); err != nil {
				panic(err)
			}
			if err := opts.validate(args); err != nil {
				panic(err)
			}
			if err := opts.run(ctx, cancel); err != nil {
				panic(err)
			}
		},
	}

	opts.config, err = opts.applyDefaults(opts.config)
	if err != nil {
		panic(err)
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

// Gardener represents all the parameters required to start the
// Gardener controller manager.
type Gardener struct {
	Config                 *config.ControllerManagerConfiguration
	K8sGardenClient        kubernetes.Interface
	K8sGardenCoreInformers gardencoreinformers.SharedInformerFactory
	KubeInformerFactory    informers.SharedInformerFactory
	Logger                 *logrus.Logger
	Recorder               record.EventRecorder
	LeaderElection         *leaderelection.LeaderElectionConfig
}

func discoveryFromControllerManagerConfiguration(cfg *config.ControllerManagerConfiguration) (discovery.CachedDiscoveryInterface, error) {
	restConfig, err := kubernetes.RESTConfigFromClientConnectionConfiguration(&cfg.GardenClientConnection, nil)
	if err != nil {
		return nil, err
	}

	discoveryCfg := cfg.Discovery
	var discoveryCacheDir string
	if discoveryCfg.DiscoveryCacheDir != nil {
		discoveryCacheDir = *discoveryCfg.DiscoveryCacheDir
	}

	var httpCacheDir string
	if discoveryCfg.HTTPCacheDir != nil {
		httpCacheDir = *discoveryCfg.HTTPCacheDir
	}

	var ttl time.Duration
	if discoveryCfg.TTL != nil {
		ttl = discoveryCfg.TTL.Duration
	}

	return diskcache.NewCachedDiscoveryClientForConfig(restConfig, discoveryCacheDir, httpCacheDir, ttl)
}

// NewGardener is the main entry point of instantiating a new Gardener controller manager.
func NewGardener(cfg *config.ControllerManagerConfiguration) (*Gardener, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	// Initialize logger
	logger := logger.NewLogger(cfg.LogLevel)
	logger.Info("Starting Gardener controller manager...")
	logger.Infof("Feature Gates: %s", features.FeatureGate.String())

	if flag := flag.Lookup("v"); flag != nil {
		if err := flag.Value.Set(fmt.Sprintf("%d", cfg.KubernetesLogLevel)); err != nil {
			return nil, err
		}
	}

	// Prepare a Kubernetes client object for the Garden cluster which contains all the Clientsets
	// that can be used to access the Kubernetes API.
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		cfg.GardenClientConnection.Kubeconfig = kubeconfig
	}

	restCfg, err := kubernetes.RESTConfigFromClientConnectionConfiguration(&cfg.GardenClientConnection, nil)
	if err != nil {
		return nil, err
	}

	disc, err := discoveryFromControllerManagerConfiguration(cfg)
	if err != nil {
		return nil, err
	}

	k8sGardenClient, err := kubernetes.NewWithConfig(
		kubernetes.WithRESTConfig(restCfg),
		kubernetes.WithClientOptions(
			client.Options{
				Mapper: restmapper.NewDeferredDiscoveryRESTMapper(disc),
				Scheme: kubernetes.GardenScheme,
			}),
	)
	if err != nil {
		return nil, err
	}
	k8sGardenClientLeaderElection, err := k8s.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	// Set up leader election if enabled and prepare event recorder.
	var (
		leaderElectionConfig *leaderelection.LeaderElectionConfig
		recorder             = utils.CreateRecorder(k8sGardenClient.Kubernetes(), "gardener-controller-manager")
	)
	if cfg.LeaderElection.LeaderElect {
		leaderElectionConfig, err = utils.MakeLeaderElectionConfig(cfg.LeaderElection.LeaderElectionConfiguration, cfg.LeaderElection.LockObjectNamespace, cfg.LeaderElection.LockObjectName, k8sGardenClientLeaderElection, recorder)
		if err != nil {
			return nil, err
		}
	}

	return &Gardener{
		Config:                 cfg,
		Logger:                 logger,
		Recorder:               recorder,
		K8sGardenClient:        k8sGardenClient,
		K8sGardenCoreInformers: gardencoreinformers.NewSharedInformerFactory(k8sGardenClient.GardenCore(), 0),
		KubeInformerFactory:    kubeinformers.NewSharedInformerFactory(k8sGardenClient.Kubernetes(), 0),
		LeaderElection:         leaderElectionConfig,
	}, nil
}

func (g *Gardener) cleanup() {
	if err := os.RemoveAll(controllermanagerconfigv1alpha1.DefaultDiscoveryDir); err != nil {
		g.Logger.Errorf("Could not cleanup base discovery cache directory: %v", err)
	}
}

// Run runs the Gardener. This should never exit.
func (g *Gardener) Run(ctx context.Context, cancel context.CancelFunc) error {
	defer g.cleanup()
	leaderElectionCtx, leaderElectionCancel := context.WithCancel(context.Background())

	// Prepare a reusable run function.
	run := func(ctx context.Context) {
		g.startControllers(ctx)
	}

	// Start HTTP server
	var (
		projectInformer = g.K8sGardenCoreInformers.Core().V1alpha1().Projects()
		shootInformer   = g.K8sGardenCoreInformers.Core().V1alpha1().Shoots()

		httpsHandlers = map[string]func(http.ResponseWriter, *http.Request){
			"/webhooks/validate-namespace-deletion": webhooks.NewValidateNamespaceDeletionHandler(g.K8sGardenClient, projectInformer.Lister(), shootInformer.Lister()),
			"/webhooks/validate-kubeconfig-secrets": webhooks.NewValidateKubeconfigSecretsHandler(),
		}
	)

	go server.ServeHTTP(ctx, g.Config.Server.HTTP.Port, g.Config.Server.HTTP.BindAddress)
	go server.ServeHTTPS(ctx, g.K8sGardenCoreInformers, httpsHandlers, g.Config.Server.HTTPS.Port, g.Config.Server.HTTPS.BindAddress, g.Config.Server.HTTPS.TLS.ServerCertPath, g.Config.Server.HTTPS.TLS.ServerKeyPath, shootInformer.Informer(), projectInformer.Informer())
	handlers.UpdateHealth(true)

	// If leader election is enabled, run via LeaderElector until done and exit.
	if g.LeaderElection != nil {
		g.LeaderElection.Callbacks = leaderelection.LeaderCallbacks{
			OnStartedLeading: func(_ context.Context) {
				g.Logger.Info("Acquired leadership, starting controllers.")
				run(ctx)
				leaderElectionCancel()
			},
			OnStoppedLeading: func() {
				g.Logger.Info("Lost leadership, terminating.")
				cancel()
			},
		}
		leaderElector, err := leaderelection.NewLeaderElector(*g.LeaderElection)
		if err != nil {
			return fmt.Errorf("couldn't create leader elector: %v", err)
		}
		leaderElector.Run(leaderElectionCtx)
		return nil
	}

	// Leader election is disabled, thus run directly until done.
	leaderElectionCancel()
	run(ctx)
	return nil
}

func (g *Gardener) startControllers(ctx context.Context) {
	controller.NewGardenControllerFactory(
		g.K8sGardenClient,
		g.K8sGardenCoreInformers,
		g.KubeInformerFactory,
		g.Config,
		g.Recorder,
	).Run(ctx)
}
