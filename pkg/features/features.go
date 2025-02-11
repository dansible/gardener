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

package features

import (
	utilfeature "k8s.io/apiserver/pkg/util/feature"
)

const (
	// Every feature gate should add method here following this template:
	//
	// // MyFeature enable Foo.
	// // owner: @username
	// // alpha: v5.X
	// MyFeature utilfeature.Feature = "MyFeature"

	// Logging enables logging stack for clusters.
	// owner @mvladev, @ialidzhikov
	// alpha: v0.13.0
	Logging utilfeature.Feature = "Logging"

	// HVPA enables simultaneous horizontal and vertical scaling in Seed Clusters.
	// owner @ggaurav10, @amshuman-kr
	// alpha: v0.1.0
	HVPA utilfeature.Feature = "HVPA"

	// HVPAForShootedSeed enables simultaneous horizontal and vertical scaling in shooted seed Clusters.
	// owner @ggaurav10, @amshuman-kr
	// alpha: v0.1.0
	HVPAForShootedSeed utilfeature.Feature = "HVPAForShootedSeed"
)
