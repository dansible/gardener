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

package quota

import (
	"context"
	"errors"
	"fmt"
	"time"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	v1alpha1constants "github.com/gardener/gardener/pkg/apis/core/v1alpha1/constants"
	gardencoreinformers "github.com/gardener/gardener/pkg/client/core/informers/externalversions"
	gardencorelisters "github.com/gardener/gardener/pkg/client/core/listers/core/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/logger"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *Controller) quotaAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Logger.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}
	c.quotaQueue.Add(key)
}

func (c *Controller) quotaUpdate(oldObj, newObj interface{}) {
	c.quotaAdd(newObj)
}

func (c *Controller) quotaDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Logger.Errorf("Couldn't get key for object %+v: %v", obj, err)
		return
	}
	c.quotaQueue.Add(key)
}

func (c *Controller) reconcileQuotaKey(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	quota, err := c.quotaLister.Quotas(namespace).Get(name)
	if apierrors.IsNotFound(err) {
		logger.Logger.Debugf("[QUOTA RECONCILE] %s - skipping because Quota has been deleted", key)
		return nil
	}
	if err != nil {
		logger.Logger.Infof("[QUOTA RECONCILE] %s - unable to retrieve object from store: %v", key, err)
		return err
	}

	if err := c.control.ReconcileQuota(quota, key); err != nil {
		c.quotaQueue.AddAfter(key, time.Minute)
	}
	return nil
}

// ControlInterface implements the control logic for updating Quotas. It is implemented as an interface to allow
// for extensions that provide different semantics. Currently, there is only one implementation.
type ControlInterface interface {
	// ReconcileQuota implements the control logic for Quota creation, update, and deletion.
	// If an implementation returns a non-nil error, the invocation will be retried using a rate-limited strategy.
	// Implementors should sink any errors that they do not wish to trigger a retry, and they may feel free to
	// exit exceptionally at any point provided they wish the update to be re-run at a later point in time.
	ReconcileQuota(quota *gardencorev1alpha1.Quota, key string) error
}

// NewDefaultControl returns a new instance of the default implementation ControlInterface that
// implements the documented semantics for Quotas. You should use an instance returned from NewDefaultControl()
// for any scenario other than testing.
func NewDefaultControl(k8sGardenClient kubernetes.Interface, k8sGardenCoreInformers gardencoreinformers.SharedInformerFactory, recorder record.EventRecorder, secretBindingLister gardencorelisters.SecretBindingLister) ControlInterface {
	return &defaultControl{k8sGardenClient, k8sGardenCoreInformers, recorder, secretBindingLister}
}

type defaultControl struct {
	k8sGardenClient        kubernetes.Interface
	k8sGardenCoreInformers gardencoreinformers.SharedInformerFactory
	recorder               record.EventRecorder
	secretBindingLister    gardencorelisters.SecretBindingLister
}

func (c *defaultControl) ReconcileQuota(obj *gardencorev1alpha1.Quota, key string) error {
	_, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return err
	}

	var (
		quota       = obj.DeepCopy()
		quotaLogger = logger.NewFieldLogger(logger.Logger, "quota", fmt.Sprintf("%s/%s", quota.Namespace, quota.Name))
	)

	// The deletionTimestamp labels a Quota as intended to get deleted. Before deletion,
	// it has to be ensured that no SecretBindings are depending on the Quota anymore.
	// When this happens the controller will remove the finalizers from the Quota so that it can be garbage collected.
	if quota.DeletionTimestamp != nil {
		if !sets.NewString(quota.Finalizers...).Has(gardencorev1alpha1.GardenerName) {
			return nil
		}

		associatedSecretBindings, err := controllerutils.DetermineSecretBindingAssociations(quota, c.secretBindingLister)
		if err != nil {
			quotaLogger.Error(err.Error())
			return err
		}

		if len(associatedSecretBindings) == 0 {
			quotaLogger.Info("No SecretBindings are referencing the Quota. Deletion accepted.")

			// Remove finalizer from Quota
			quotaFinalizers := sets.NewString(quota.Finalizers...)
			quotaFinalizers.Delete(gardencorev1alpha1.GardenerName)
			quota.Finalizers = quotaFinalizers.UnsortedList()
			if _, err := c.k8sGardenClient.GardenCore().CoreV1alpha1().Quotas(quota.Namespace).Update(quota); client.IgnoreNotFound(err) != nil {
				quotaLogger.Error(err.Error())
				return err
			}
			return nil
		}

		message := fmt.Sprintf("Can't delete Quota, because the following SecretBindings are still referencing it: %v", associatedSecretBindings)
		quotaLogger.Info(message)
		c.recorder.Event(quota, corev1.EventTypeNormal, v1alpha1constants.EventResourceReferenced, message)

		return errors.New("Quota still has references")
	}

	if err := controllerutils.EnsureFinalizer(context.TODO(), c.k8sGardenClient.Client(), quota, gardencorev1alpha1.GardenerName); err != nil {
		quotaLogger.Errorf("Could not add finalizer to Quota: %s", err.Error())
		return err
	}

	return nil
}
