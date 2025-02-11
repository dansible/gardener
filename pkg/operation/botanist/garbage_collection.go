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

package botanist

import (
	"context"
	"strings"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/pkg/utils"

	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	"github.com/hashicorp/go-multierror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PerformGarbageCollectionSeed performs garbage collection in the Shoot namespace in the Seed cluster,
// i.e., it deletes old machine sets which have a desired=actual=0 replica count.
func (b *Botanist) PerformGarbageCollectionSeed() error {
	ctx := context.TODO()

	podList := &corev1.PodList{}
	if err := b.K8sSeedClient.Client().List(ctx, podList, client.InNamespace(b.Shoot.SeedNamespace)); err != nil {
		return err
	}

	if err := b.deleteStalePods(b.K8sSeedClient.Client(), podList); err != nil {
		return err
	}

	machineSetList := &machinev1alpha1.MachineSetList{}
	if err := b.K8sSeedClient.Client().List(ctx, machineSetList, client.InNamespace(b.Shoot.SeedNamespace)); err != nil {
		return err
	}

	for _, machineSet := range machineSetList.Items {
		if machineSet.Spec.Replicas == 0 && machineSet.Status.Replicas == 0 {
			b.Logger.Debugf("Deleting MachineSet %s as the number of desired and actual replicas is 0.", machineSet.Name)
			if err := b.K8sSeedClient.Client().Delete(ctx, machineSet.DeepCopy()); client.IgnoreNotFound(err) != nil {
				return err
			}
		}
	}
	return nil
}

// PerformGarbageCollectionShoot performs garbage collection in the kube-system namespace in the Shoot
// cluster, i.e., it deletes evicted pods (mitigation for https://github.com/kubernetes/kubernetes/issues/55051).
func (b *Botanist) PerformGarbageCollectionShoot() error {
	// Workaround for https://github.com/kubernetes/kubernetes/pull/72507.
	if err := b.removeStaleOutOfDiskNodeCondition(); err != nil {
		return err
	}

	namespace := metav1.NamespaceSystem
	if b.Shoot.Info.DeletionTimestamp != nil {
		namespace = metav1.NamespaceAll
	}

	podList := &corev1.PodList{}
	if err := b.K8sShootClient.Client().List(context.TODO(), podList, client.InNamespace(namespace)); err != nil {
		return err
	}

	return b.deleteStalePods(b.K8sShootClient.Client(), podList)
}

func (b *Botanist) deleteStalePods(k8sClient client.Client, podList *corev1.PodList) error {
	var result error

	for _, pod := range podList.Items {
		if strings.Contains(pod.Status.Reason, "Evicted") {
			b.Logger.Debugf("Deleting pod %s as its reason is %s.", pod.Name, pod.Status.Reason)
			if err := k8sClient.Delete(context.TODO(), &pod, kubernetes.DefaultDeleteOptions...); client.IgnoreNotFound(err) != nil {
				result = multierror.Append(result, err)
			}
			continue
		}

		if common.ShouldObjectBeRemoved(&pod, common.GardenerDeletionGracePeriod) {
			b.Logger.Debugf("Deleting stuck terminating pod %q", pod.Name)
			if err := k8sClient.Delete(context.TODO(), &pod, kubernetes.ForceDeleteOptions...); client.IgnoreNotFound(err) != nil {
				result = multierror.Append(result, err)
			}
		}
	}

	return result
}

func (b *Botanist) removeStaleOutOfDiskNodeCondition() error {
	// This code is limited to 1.13.0-1.13.3 (1.13.4 contains the Kubernetes fix).
	// For more details see https://github.com/kubernetes/kubernetes/pull/73394.
	needsRemovalOfStaleCondition, err := utils.CheckVersionMeetsConstraint(b.Shoot.Info.Spec.Kubernetes.Version, ">= 1.13.0, <= 1.13.3")
	if err != nil {
		return err
	}
	if !needsRemovalOfStaleCondition {
		return nil
	}

	nodeList := &corev1.NodeList{}
	if err := b.K8sShootClient.Client().List(context.TODO(), nodeList); err != nil {
		return err
	}

	var result error
	for _, node := range nodeList.Items {
		var conditions []corev1.NodeCondition

		for _, condition := range node.Status.Conditions {
			if condition.Type != corev1.NodeOutOfDisk {
				conditions = append(conditions, condition)
			}
		}

		if len(conditions) == len(node.Status.Conditions) {
			continue
		}

		node.Status.Conditions = conditions

		if err := b.K8sShootClient.Client().Status().Update(context.TODO(), &node); err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result
}
