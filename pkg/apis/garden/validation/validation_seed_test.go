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

package validation_test

import (
	"github.com/gardener/gardener/pkg/apis/garden"
	. "github.com/gardener/gardener/pkg/apis/garden/validation"
	"github.com/gardener/gardener/pkg/operation/common"
	. "github.com/gardener/gardener/pkg/utils/validation/gomega"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("Seed Validation Tests", func() {
	Describe("#ValidateSeed, #ValidateSeedUpdate", func() {
		var (
			seed   *garden.Seed
			backup *garden.SeedBackup
		)

		BeforeEach(func() {
			region := "some-region"
			pods := "10.240.0.0/16"
			services := "10.241.0.0/16"
			seed = &garden.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "seed-1",
					Annotations: map[string]string{
						common.AnnotatePersistentVolumeMinimumSize: "10Gi",
					},
				},
				Spec: garden.SeedSpec{
					Cloud: garden.SeedCloud{
						Profile: "aws",
						Region:  "eu-west-1",
					},
					Provider: garden.SeedProvider{
						Type:   "aws",
						Region: "eu-west-1",
					},
					IngressDomain: "ingress.my-seed-1.example.com",
					SecretRef: &corev1.SecretReference{
						Name:      "seed-aws",
						Namespace: "garden",
					},
					Taints: []garden.SeedTaint{
						{Key: garden.SeedTaintProtected},
					},
					Networks: garden.SeedNetworks{
						Nodes:    "10.250.0.0/16",
						Pods:     "100.96.0.0/11",
						Services: "100.64.0.0/13",
						ShootDefaults: &garden.ShootNetworks{
							Pods:     &pods,
							Services: &services,
						},
					},
					Backup: &garden.SeedBackup{
						Provider: garden.CloudProviderAWS,
						Region:   &region,
						SecretRef: corev1.SecretReference{
							Name:      "backup-aws",
							Namespace: "garden",
						},
					},
				},
			}
		})

		It("should not return any errors", func() {
			errorList := ValidateSeed(seed)

			Expect(errorList).To(HaveLen(0))
		})

		It("should forbid Seed resources with empty metadata", func() {
			seed.ObjectMeta = metav1.ObjectMeta{}

			errorList := ValidateSeed(seed)

			Expect(errorList).To(HaveLen(1))
			Expect(*errorList[0]).To(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.name"),
			}))
		})

		It("should forbid invalid annotations", func() {
			seed.ObjectMeta.Annotations = map[string]string{
				common.AnnotatePersistentVolumeMinimumSize: "10Gix",
			}
			errorList := ValidateSeed(seed)
			Expect(errorList).To(HaveLen(1))
		})

		It("should forbid Seed specification with empty or invalid keys", func() {
			invalidCIDR := "invalid-cidr"
			seed.Spec.Cloud = garden.SeedCloud{}
			seed.Spec.Provider = garden.SeedProvider{}
			seed.Spec.IngressDomain = "invalid_dns1123-subdomain"
			seed.Spec.SecretRef = &corev1.SecretReference{}
			seed.Spec.Networks = garden.SeedNetworks{
				Nodes:    invalidCIDR,
				Pods:     "300.300.300.300/300",
				Services: invalidCIDR,
				ShootDefaults: &garden.ShootNetworks{
					Pods:     &invalidCIDR,
					Services: &invalidCIDR,
				},
			}
			seed.Spec.Taints = []garden.SeedTaint{
				{Key: garden.SeedTaintProtected},
				{Key: garden.SeedTaintProtected},
				{Key: ""},
			}
			seed.Spec.Backup.SecretRef = corev1.SecretReference{}
			seed.Spec.Backup.Provider = ""
			minSize := resource.MustParse("-1")
			seed.Spec.Volume = &garden.SeedVolume{
				MinimumSize: &minSize,
				Providers: []garden.SeedVolumeProvider{
					{
						Purpose: "",
						Name:    "",
					},
					{
						Purpose: "duplicate",
						Name:    "value1",
					},
					{
						Purpose: "duplicate",
						Name:    "value2",
					},
				},
			}

			errorList := ValidateSeed(seed)

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("spec.backup.provider"),
					"Detail": Equal(`must provide a backup cloud provider name`),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("spec.backup.secretRef.name"),
					"Detail": Equal(`must provide a name`),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("spec.backup.secretRef.namespace"),
					"Detail": Equal(`must provide a namespace`),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.cloud.profile"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.cloud.region"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.provider.type"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.provider.region"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.ingressDomain"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.secretRef.name"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.secretRef.namespace"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("spec.taints[1].key"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.taints[2].key"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeNotSupported),
					"Field": Equal("spec.taints[2].key"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.networks.nodes"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.networks.pods"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.networks.services"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.networks.shootDefaults.pods"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.networks.shootDefaults.services"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.volume.minimumSize"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.volume.providers[0].purpose"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeRequired),
					"Field": Equal("spec.volume.providers[0].name"),
				})),
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeDuplicate),
					"Field": Equal("spec.volume.providers[2].purpose"),
				})),
			))
		})

		It("should forbid Seed with overlapping networks", func() {
			shootDefaultPodCIDR := "10.0.1.128/28"     // 10.0.1.128 -> 10.0.1.13
			shootDefaultServiceCIDR := "10.0.1.144/30" // 10.0.1.144 -> 10.0.1.17
			// Pods CIDR overlaps with Nodes network
			// Services CIDR overlaps with Nodes and Pods
			// Shoot default pod CIDR overlaps with services
			// Shoot default pod CIDR overlaps with shoot default pod CIDR
			seed.Spec.Networks = garden.SeedNetworks{
				Nodes:    "10.0.0.0/8",   // 10.0.0.0 -> 10.255.255.25
				Pods:     "10.0.1.0/24",  // 10.0.1.0 -> 10.0.1.25
				Services: "10.0.1.64/26", // 10.0.1.64 -> 10.0.1.17
				ShootDefaults: &garden.ShootNetworks{
					Pods:     &shootDefaultPodCIDR,
					Services: &shootDefaultServiceCIDR,
				},
			}

			errorList := ValidateSeed(seed)

			Expect(errorList).To(ConsistOfFields(Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.networks.pods"),
				"Detail": Equal(`must not be a subset of "spec.networks.nodes" ("10.0.0.0/8")`),
			}, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.networks.services"),
				"Detail": Equal(`must not be a subset of "spec.networks.nodes" ("10.0.0.0/8")`),
			}, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.networks.shootDefaults.pods"),
				"Detail": Equal(`must not be a subset of "spec.networks.nodes" ("10.0.0.0/8")`),
			}, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.networks.shootDefaults.services"),
				"Detail": Equal(`must not be a subset of "spec.networks.nodes" ("10.0.0.0/8")`),
			}, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.networks.services"),
				"Detail": Equal(`must not be a subset of "spec.networks.pods" ("10.0.1.0/24")`),
			}, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.networks.shootDefaults.pods"),
				"Detail": Equal(`must not be a subset of "spec.networks.pods" ("10.0.1.0/24")`),
			}, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.networks.shootDefaults.services"),
				"Detail": Equal(`must not be a subset of "spec.networks.pods" ("10.0.1.0/24")`),
			}))
		})

		It("should fail updating immutable fields", func() {
			newSeed := prepareSeedForUpdate(seed)
			newSeed.Spec.Networks = garden.SeedNetworks{
				Nodes:    "10.1.0.0/16",
				Pods:     "10.2.0.0/16",
				Services: "10.3.1.64/26",
			}
			otherRegion := "other-region"
			newSeed.Spec.Backup.Provider = "other-provider"
			newSeed.Spec.Backup.Region = &otherRegion

			errorList := ValidateSeedUpdate(newSeed, seed)

			Expect(errorList).To(ConsistOfFields(Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.networks.pods"),
				"Detail": Equal(`field is immutable`),
			}, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.networks.services"),
				"Detail": Equal(`field is immutable`),
			}, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.networks.nodes"),
				"Detail": Equal(`field is immutable`),
			}, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.backup.region"),
				"Detail": Equal(`field is immutable`),
			}, Fields{
				"Type":   Equal(field.ErrorTypeInvalid),
				"Field":  Equal("spec.backup.provider"),
				"Detail": Equal(`field is immutable`),
			}))
		})

		Context("#validateSeedBackupUpdate", func() {
			It("should allow adding backup profile", func() {
				seed.Spec.Backup = nil
				newSeed := prepareSeedForUpdate(seed)
				newSeed.Spec.Backup = backup

				errorList := ValidateSeedUpdate(newSeed, seed)

				Expect(errorList).To(BeEmpty())
			})

			It("should forbid removing backup profile", func() {
				newSeed := prepareSeedForUpdate(seed)
				newSeed.Spec.Backup = nil

				errorList := ValidateSeedUpdate(newSeed, seed)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("spec.backup"),
					"Detail": Equal(`field is immutable`),
				}))
			})
		})
	})
})

func prepareSeedForUpdate(seed *garden.Seed) *garden.Seed {
	s := seed.DeepCopy()
	s.ResourceVersion = "1"
	return s
}
