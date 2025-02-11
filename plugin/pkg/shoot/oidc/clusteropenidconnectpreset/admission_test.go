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

package clusteropenidconnectpreset_test

import (
	"github.com/gardener/gardener/pkg/apis/garden"
	settingsv1alpha1 "github.com/gardener/gardener/pkg/apis/settings/v1alpha1"
	gardeninformers "github.com/gardener/gardener/pkg/client/garden/informers/internalversion"
	settingsinformer "github.com/gardener/gardener/pkg/client/settings/informers/externalversions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/utils/pointer"

	. "github.com/gardener/gardener/plugin/pkg/shoot/oidc/clusteropenidconnectpreset"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cluster OpenIDConfig Preset", func() {
	Describe("#Admit", func() {
		var (
			admissionHandler        *ClusterOpenIDConnectPreset
			settingsInformerFactory settingsinformer.SharedInformerFactory
			shoot                   *garden.Shoot
			project                 *garden.Project
			preset                  *settingsv1alpha1.ClusterOpenIDConnectPreset
			gardenInformerFactory   gardeninformers.SharedInformerFactory
		)

		BeforeEach(func() {
			namespace := "my-namespace"
			shootName := "shoot"
			presetName := "preset-1"
			projectName := "project-1"
			shoot = &garden.Shoot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      shootName,
					Namespace: namespace,
				},
				Spec: garden.ShootSpec{
					Kubernetes: garden.Kubernetes{
						Version: "1.15",
					},
				},
			}

			project = &garden.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: projectName,
				},
				Spec: garden.ProjectSpec{
					Namespace: pointer.StringPtr(namespace),
				},
				Status: garden.ProjectStatus{
					Phase: garden.ProjectReady,
				},
			}

			preset = &settingsv1alpha1.ClusterOpenIDConnectPreset{
				ObjectMeta: metav1.ObjectMeta{
					Name:      presetName,
					Namespace: namespace,
				},
				Spec: settingsv1alpha1.ClusterOpenIDConnectPresetSpec{
					ProjectSelector: &metav1.LabelSelector{},
					OpenIDConnectPresetSpec: settingsv1alpha1.OpenIDConnectPresetSpec{
						// select everything
						ShootSelector: &metav1.LabelSelector{},
						Weight:        0,
						Server: settingsv1alpha1.KubeAPIServerOpenIDConnect{
							CABundle:     pointer.StringPtr("cert"),
							ClientID:     "client-id",
							IssuerURL:    "https://foo.bar",
							GroupsClaim:  pointer.StringPtr("groupz"),
							GroupsPrefix: pointer.StringPtr("group-prefix"),
							RequiredClaims: map[string]string{
								"claim-1": "value-1",
								"claim-2": "value-2",
							},
							SigningAlgs:    []string{"alg-1", "alg-2"},
							UsernameClaim:  pointer.StringPtr("user"),
							UsernamePrefix: pointer.StringPtr("user-prefix"),
						},
						Client: &settingsv1alpha1.OpenIDConnectClientAuthentication{
							Secret:      pointer.StringPtr("secret"),
							ExtraConfig: map[string]string{"foo": "bar", "baz": "dap"},
						},
					},
				},
			}
			admissionHandler, _ = New()
			admissionHandler.AssignReadyFunc(func() bool { return true })
			settingsInformerFactory = settingsinformer.NewSharedInformerFactory(nil, 0)
			admissionHandler.SetSettingsInformerFactory(settingsInformerFactory)
			gardenInformerFactory = gardeninformers.NewSharedInformerFactory(nil, 0)
			admissionHandler.SetInternalGardenInformerFactory(gardenInformerFactory)

		})

		Context("should do nothing when", func() {

			var (
				expected *garden.Shoot
				attrs    admission.Attributes
			)

			BeforeEach(func() {
				expected = shoot.DeepCopy()
				attrs = admission.NewAttributesRecord(shoot, nil, garden.Kind("Shoot").WithVersion("v1beta1"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("v1beta1"), "", admission.Create, false, nil)
			})

			AfterEach(func() {
				Expect(admissionHandler.Admit(attrs, nil)).NotTo(HaveOccurred())
				Expect(shoot).To(Equal(expected))
			})

			DescribeTable("disabled operations",
				func(op admission.Operation) {
					attrs = admission.NewAttributesRecord(shoot, nil, garden.Kind("Shoot").WithVersion("v1beta1"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("v1beta1"), "", op, false, nil)
				},
				Entry("update verb", admission.Update),
				Entry("delete verb", admission.Delete),
				Entry("connect verb", admission.Connect),
			)

			It("subresource is status", func() {
				attrs = admission.NewAttributesRecord(shoot, nil, garden.Kind("Shoot").WithVersion("v1beta1"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("v1beta1"), "status", admission.Create, false, nil)
			})

			It("version is not correct", func() {
				shoot.Spec.Kubernetes.Version = "something-not-valid"
				expected = shoot.DeepCopy()
			})

			It("preset shoot label selector doesn't match", func() {
				preset.Spec.ShootSelector.MatchLabels = map[string]string{"not": "existing"}
				settingsInformerFactory.Settings().V1alpha1().ClusterOpenIDConnectPresets().Informer().GetStore().Add(preset)
				gardenInformerFactory.Garden().InternalVersion().Projects().Informer().GetStore().Add(project)
			})

			It("preset preset label selector doesn't match", func() {
				preset.Spec.ProjectSelector.MatchLabels = map[string]string{"not": "existing"}
				settingsInformerFactory.Settings().V1alpha1().ClusterOpenIDConnectPresets().Informer().GetStore().Add(preset)
				gardenInformerFactory.Garden().InternalVersion().Projects().Informer().GetStore().Add(project)
			})

			It("oidc settings already exist", func() {
				shoot.Spec.Kubernetes.KubeAPIServer = &garden.KubeAPIServerConfig{
					OIDCConfig: &garden.OIDCConfig{},
				}
				expected = shoot.DeepCopy()
			})
		})

		Context("should return error", func() {
			var (
				expected     *garden.Shoot
				attrs        admission.Attributes
				errorFunc    func(error) bool
				errorMessage string
			)

			BeforeEach(func() {
				expected = shoot.DeepCopy()
				attrs = admission.NewAttributesRecord(shoot, nil, garden.Kind("Shoot").WithVersion("v1beta1"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("v1beta1"), "", admission.Create, false, nil)
				errorFunc = nil
				errorMessage = ""
			})

			AfterEach(func() {
				err := admissionHandler.Admit(attrs, nil)
				Expect(err).To(HaveOccurred())
				Expect(errorFunc(err)).To(BeTrue(), "error type should be the same")
				Expect(shoot).To(Equal(expected))
				Expect(err.Error()).To(Equal(errorMessage))
			})

			It("when received not a Shoot object", func() {
				attrs = admission.NewAttributesRecord(&garden.Seed{}, nil, garden.Kind("Shoot").WithVersion("v1beta1"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("v1beta1"), "", admission.Create, false, nil)
				errorFunc = apierrors.IsBadRequest
				errorMessage = "could not convert resource into Shoot object"
			})

			It("when it's not ready", func() {
				Skip("this takes 10seconds to pass")
				admissionHandler.AssignReadyFunc(func() bool { return false })
				errorFunc = apierrors.IsForbidden
				errorMessage = `presets.garden.sapcloud.io "shoot" is forbidden: not yet ready to handle request`
			})

		})

		Context("should mutate the result", func() {
			var (
				expected *garden.Shoot
			)

			BeforeEach(func() {
				expected = shoot.DeepCopy()
				expected.Spec.Kubernetes.KubeAPIServer = &garden.KubeAPIServerConfig{
					OIDCConfig: &garden.OIDCConfig{
						CABundle:     pointer.StringPtr("cert"),
						ClientID:     pointer.StringPtr("client-id"),
						IssuerURL:    pointer.StringPtr("https://foo.bar"),
						GroupsClaim:  pointer.StringPtr("groupz"),
						GroupsPrefix: pointer.StringPtr("group-prefix"),
						RequiredClaims: map[string]string{
							"claim-1": "value-1",
							"claim-2": "value-2",
						},
						SigningAlgs:    []string{"alg-1", "alg-2"},
						UsernameClaim:  pointer.StringPtr("user"),
						UsernamePrefix: pointer.StringPtr("user-prefix"),

						ClientAuthentication: &garden.OpenIDConnectClientAuthentication{
							Secret:      pointer.StringPtr("secret"),
							ExtraConfig: map[string]string{"foo": "bar", "baz": "dap"},
						},
					},
				}
			})

			AfterEach(func() {
				settingsInformerFactory.Settings().V1alpha1().ClusterOpenIDConnectPresets().Informer().GetStore().Add(preset)
				gardenInformerFactory.Garden().InternalVersion().Projects().Informer().GetStore().Add(project)

				attrs := admission.NewAttributesRecord(shoot, nil, garden.Kind("Shoot").WithVersion("v1beta1"), shoot.Namespace, shoot.Name, garden.Resource("shoots").WithVersion("v1alpha1"), "", admission.Create, false, nil)
				err := admissionHandler.Admit(attrs, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(shoot.Spec.Kubernetes.KubeAPIServer).NotTo(BeNil())
				Expect(shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig).NotTo(BeNil())
				Expect(shoot.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientAuthentication).NotTo(BeNil())
				Expect(shoot).To(Equal(expected))
			})

			It("shoot's kube-apiserver-oidc settings is not set", func() {
				shoot.Spec.Kubernetes.KubeAPIServer = &garden.KubeAPIServerConfig{}
			})

			It("successfully", func() {
			})

			It("presets which match even with lower weight", func() {
				preset2 := preset.DeepCopy()

				preset.Spec.OpenIDConnectPresetSpec.Weight = 100
				preset.Spec.OpenIDConnectPresetSpec.ShootSelector.MatchLabels = map[string]string{"not": "existing"}

				preset2.ObjectMeta.Name = "preset-2"
				preset2.Spec.OpenIDConnectPresetSpec.Server.ClientID = "client-id-2"

				expected.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientID = pointer.StringPtr("client-id-2")

				settingsInformerFactory.Settings().V1alpha1().ClusterOpenIDConnectPresets().Informer().GetStore().Add(preset2)
			})

			It("preset with higher weight", func() {
				preset2 := preset.DeepCopy()
				preset2.ObjectMeta.Name = "preset-2"
				preset2.Spec.OpenIDConnectPresetSpec.Weight = 100
				preset2.Spec.OpenIDConnectPresetSpec.Server.ClientID = "client-id-2"

				expected.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientID = pointer.StringPtr("client-id-2")

				settingsInformerFactory.Settings().V1alpha1().ClusterOpenIDConnectPresets().Informer().GetStore().Add(preset2)
			})

			It("presets ordered lexicographically", func() {
				preset.ObjectMeta.Name = "01preset"
				preset2 := preset.DeepCopy()
				preset2.ObjectMeta.Name = "02preset"
				preset2.Spec.OpenIDConnectPresetSpec.Server.ClientID = "client-id-2"

				expected.Spec.Kubernetes.KubeAPIServer.OIDCConfig.ClientID = pointer.StringPtr("client-id-2")

				settingsInformerFactory.Settings().V1alpha1().ClusterOpenIDConnectPresets().Informer().GetStore().Add(preset2)
			})

			It("presets which don't match the project selector", func() {
				preset2 := preset.DeepCopy()
				preset2.Spec.ProjectSelector.MatchLabels = map[string]string{"not": "existing"}
				preset2.Spec.OpenIDConnectPresetSpec.Weight = 100
				preset2.Spec.OpenIDConnectPresetSpec.Server.ClientID = "client-id-2"

				settingsInformerFactory.Settings().V1alpha1().ClusterOpenIDConnectPresets().Informer().GetStore().Add(preset2)
			})

			It("remove required claims for K8S < 1.11", func() {
				shoot.Spec.Kubernetes.Version = "1.10"

				expected.Spec.Kubernetes.Version = "1.10"
				expected.Spec.Kubernetes.KubeAPIServer.OIDCConfig.RequiredClaims = nil
			})
		})
	})

	Describe("#ValidateInitialization", func() {

		var (
			plugin *ClusterOpenIDConnectPreset
		)

		BeforeEach(func() {
			plugin = &ClusterOpenIDConnectPreset{}
		})

		Context("error occures", func() {

			It("when clusterOIDCLister is not set", func() {
				err := plugin.ValidateInitialization()
				Expect(err).To(HaveOccurred())
			})

			It("when projectLister is not set", func() {
				plugin.SetSettingsInformerFactory(settingsinformer.NewSharedInformerFactory(nil, 0))
				err := plugin.ValidateInitialization()
				Expect(err).To(HaveOccurred())
			})
		})

		It("should return nil error when everything is set", func() {
			plugin.SetSettingsInformerFactory(settingsinformer.NewSharedInformerFactory(nil, 0))
			plugin.SetInternalGardenInformerFactory(gardeninformers.NewSharedInformerFactory(nil, 0))
			Expect(plugin.ValidateInitialization()).ToNot(HaveOccurred())
		})
	})
})
