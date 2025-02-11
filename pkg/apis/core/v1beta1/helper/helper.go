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

package helper

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/logger"
	"github.com/gardener/gardener/pkg/utils"

	"github.com/Masterminds/semver"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Now determines the current metav1.Time.
var Now = metav1.Now

// InitCondition initializes a new Condition with an Unknown status.
func InitCondition(conditionType gardencorev1beta1.ConditionType) gardencorev1beta1.Condition {
	return gardencorev1beta1.Condition{
		Type:               conditionType,
		Status:             gardencorev1beta1.ConditionUnknown,
		Reason:             "ConditionInitialized",
		Message:            "The condition has been initialized but its semantic check has not been performed yet.",
		LastTransitionTime: Now(),
	}
}

// NewConditions initializes the provided conditions based on an existing list. If a condition type does not exist
// in the list yet, it will be set to default values.
func NewConditions(conditions []gardencorev1beta1.Condition, conditionTypes ...gardencorev1beta1.ConditionType) []*gardencorev1beta1.Condition {
	newConditions := []*gardencorev1beta1.Condition{}

	// We retrieve the current conditions in order to update them appropriately.
	for _, conditionType := range conditionTypes {
		if c := GetCondition(conditions, conditionType); c != nil {
			newConditions = append(newConditions, c)
			continue
		}
		initializedCondition := InitCondition(conditionType)
		newConditions = append(newConditions, &initializedCondition)
	}

	return newConditions
}

// GetCondition returns the condition with the given <conditionType> out of the list of <conditions>.
// In case the required type could not be found, it returns nil.
func GetCondition(conditions []gardencorev1beta1.Condition, conditionType gardencorev1beta1.ConditionType) *gardencorev1beta1.Condition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			c := condition
			return &c
		}
	}
	return nil
}

// GetOrInitCondition tries to retrieve the condition with the given condition type from the given conditions.
// If the condition could not be found, it returns an initialized condition of the given type.
func GetOrInitCondition(conditions []gardencorev1beta1.Condition, conditionType gardencorev1beta1.ConditionType) gardencorev1beta1.Condition {
	if condition := GetCondition(conditions, conditionType); condition != nil {
		return *condition
	}
	return InitCondition(conditionType)
}

// UpdatedCondition updates the properties of one specific condition.
func UpdatedCondition(condition gardencorev1beta1.Condition, status gardencorev1beta1.ConditionStatus, reason, message string) gardencorev1beta1.Condition {
	newCondition := gardencorev1beta1.Condition{
		Type:               condition.Type,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: condition.LastTransitionTime,
		LastUpdateTime:     Now(),
	}

	if condition.Status != status {
		newCondition.LastTransitionTime = Now()
	}
	return newCondition
}

func UpdatedConditionUnknownError(condition gardencorev1beta1.Condition, err error) gardencorev1beta1.Condition {
	return UpdatedConditionUnknownErrorMessage(condition, err.Error())
}

func UpdatedConditionUnknownErrorMessage(condition gardencorev1beta1.Condition, message string) gardencorev1beta1.Condition {
	return UpdatedCondition(condition, gardencorev1beta1.ConditionUnknown, gardencorev1beta1.ConditionCheckError, message)
}

// MergeConditions merges the given <oldConditions> with the <newConditions>. Existing conditions are superseded by
// the <newConditions> (depending on the condition type).
func MergeConditions(oldConditions []gardencorev1beta1.Condition, newConditions ...gardencorev1beta1.Condition) []gardencorev1beta1.Condition {
	var (
		out         = make([]gardencorev1beta1.Condition, 0, len(oldConditions))
		typeToIndex = make(map[gardencorev1beta1.ConditionType]int, len(oldConditions))
	)

	for i, condition := range oldConditions {
		out = append(out, condition)
		typeToIndex[condition.Type] = i
	}

	for _, condition := range newConditions {
		if index, ok := typeToIndex[condition.Type]; ok {
			out[index] = condition
			continue
		}
		out = append(out, condition)
	}

	return out
}

// ConditionsNeedUpdate returns true if the <existingConditions> must be updated based on <newConditions>.
func ConditionsNeedUpdate(existingConditions, newConditions []gardencorev1beta1.Condition) bool {
	return existingConditions == nil || !apiequality.Semantic.DeepEqual(newConditions, existingConditions)
}

// IsResourceSupported returns true if a given combination of kind/type is part of a controller resources list.
func IsResourceSupported(resources []gardencorev1beta1.ControllerResource, resourceKind, resourceType string) bool {
	for _, resource := range resources {
		if resource.Kind == resourceKind && strings.ToLower(resource.Type) == strings.ToLower(resourceType) {
			return true
		}
	}

	return false
}

// IsControllerInstallationSuccessful returns true if a ControllerInstallation has been marked as "successfully"
// installed.
func IsControllerInstallationSuccessful(controllerInstallation gardencorev1beta1.ControllerInstallation) bool {
	for _, condition := range controllerInstallation.Status.Conditions {
		if condition.Type == gardencorev1beta1.ControllerInstallationInstalled && condition.Status == gardencorev1beta1.ConditionTrue {
			return true
		}
	}

	return false
}

// ComputeOperationType checksthe <lastOperation> and determines whether is it is Create operation or reconcile operation
func ComputeOperationType(meta metav1.ObjectMeta, lastOperation *gardencorev1beta1.LastOperation) gardencorev1beta1.LastOperationType {
	switch {
	case meta.DeletionTimestamp != nil:
		return gardencorev1beta1.LastOperationTypeDelete
	case lastOperation == nil:
		return gardencorev1beta1.LastOperationTypeCreate
	case (lastOperation.Type == gardencorev1beta1.LastOperationTypeCreate && lastOperation.State != gardencorev1beta1.LastOperationStateSucceeded):
		return gardencorev1beta1.LastOperationTypeCreate
	}
	return gardencorev1beta1.LastOperationTypeReconcile
}

// TaintsHave returns true if the given key is part of the taints list.
func TaintsHave(taints []gardencorev1beta1.SeedTaint, key string) bool {
	for _, taint := range taints {
		if taint.Key == key {
			return true
		}
	}
	return false
}

type ShootedSeed struct {
	Protected         *bool
	Visible           *bool
	MinimumVolumeSize *string
	APIServer         *ShootedSeedAPIServer
	BlockCIDRs        []string
	ShootDefaults     *gardencorev1beta1.ShootNetworks
	Backup            *gardencorev1beta1.SeedBackup
}

type ShootedSeedAPIServer struct {
	Replicas   *int32
	Autoscaler *ShootedSeedAPIServerAutoscaler
}

type ShootedSeedAPIServerAutoscaler struct {
	MinReplicas *int32
	MaxReplicas int32
}

func parseInt32(s string) (int32, error) {
	i64, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(i64), nil
}

func parseShootedSeed(annotation string) (*ShootedSeed, error) {
	var (
		flags    = make(map[string]struct{})
		settings = make(map[string]string)

		trueVar  = true
		falseVar = false

		shootedSeed ShootedSeed
	)

	for _, fragment := range strings.Split(annotation, ",") {
		parts := strings.SplitN(fragment, "=", 2)
		if len(parts) == 1 {
			flags[fragment] = struct{}{}
			continue
		}

		settings[parts[0]] = parts[1]
	}

	if _, ok := flags["true"]; !ok {
		return nil, nil
	}

	apiServer, err := parseShootedSeedAPIServer(settings)
	if err != nil {
		return nil, err
	}
	shootedSeed.APIServer = apiServer

	blockCIDRs, err := parseShootedSeedBlockCIDRs(settings)
	if err != nil {
		return nil, err
	}
	shootedSeed.BlockCIDRs = blockCIDRs

	shootDefaults, err := parseShootedSeedShootDefaults(settings)
	if err != nil {
		return nil, err
	}
	shootedSeed.ShootDefaults = shootDefaults

	backup, err := parseShootedSeedBackup(settings)
	if err != nil {
		return nil, err
	}
	shootedSeed.Backup = backup

	if size, ok := settings["minimumVolumeSize"]; ok {
		shootedSeed.MinimumVolumeSize = &size
	}

	if _, ok := flags["protected"]; ok {
		shootedSeed.Protected = &trueVar
	}
	if _, ok := flags["unprotected"]; ok {
		shootedSeed.Protected = &falseVar
	}
	if _, ok := flags["visible"]; ok {
		shootedSeed.Visible = &trueVar
	}
	if _, ok := flags["invisible"]; ok {
		shootedSeed.Visible = &falseVar
	}

	return &shootedSeed, nil
}

func parseShootedSeedBlockCIDRs(settings map[string]string) ([]string, error) {
	cidrs, ok := settings["blockCIDRs"]
	if !ok {
		return nil, nil
	}

	var addresses []string
	for _, addr := range strings.Split(cidrs, ";") {
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

func parseShootedSeedShootDefaults(settings map[string]string) (*gardencorev1beta1.ShootNetworks, error) {
	var (
		podCIDR, ok1     = settings["shootDefaults.pods"]
		serviceCIDR, ok2 = settings["shootDefaults.services"]
	)

	if !ok1 && !ok2 {
		return nil, nil
	}

	shootNetworks := &gardencorev1beta1.ShootNetworks{}

	if ok1 {
		shootNetworks.Pods = &podCIDR
	}

	if ok2 {
		shootNetworks.Services = &serviceCIDR
	}

	return shootNetworks, nil
}

func parseShootedSeedBackup(settings map[string]string) (*gardencorev1beta1.SeedBackup, error) {
	var (
		provider, ok1           = settings["backup.provider"]
		region, ok2             = settings["backup.region"]
		secretRefName, ok3      = settings["backup.secretRef.name"]
		secretRefNamespace, ok4 = settings["backup.secretRef.namespace"]
	)

	if ok1 && provider == "none" {
		return nil, nil
	}

	backup := &gardencorev1beta1.SeedBackup{}

	if ok1 {
		backup.Provider = provider
	}
	if ok2 {
		backup.Region = &region
	}
	if ok3 {
		backup.SecretRef.Name = secretRefName
	}
	if ok4 {
		backup.SecretRef.Namespace = secretRefNamespace
	}

	return backup, nil
}

func parseShootedSeedAPIServer(settings map[string]string) (*ShootedSeedAPIServer, error) {
	apiServerAutoscaler, err := parseShootedSeedAPIServerAutoscaler(settings)
	if err != nil {
		return nil, err
	}

	replicasString, ok := settings["apiServer.replicas"]
	if !ok && apiServerAutoscaler == nil {
		return nil, nil
	}

	var apiServer ShootedSeedAPIServer

	apiServer.Autoscaler = apiServerAutoscaler

	if ok {
		replicas, err := parseInt32(replicasString)
		if err != nil {
			return nil, err
		}

		apiServer.Replicas = &replicas
	}

	return &apiServer, nil
}

func parseShootedSeedAPIServerAutoscaler(settings map[string]string) (*ShootedSeedAPIServerAutoscaler, error) {
	minReplicasString, ok1 := settings["apiServer.autoscaler.minReplicas"]
	maxReplicasString, ok2 := settings["apiServer.autoscaler.maxReplicas"]
	if !ok1 && !ok2 {
		return nil, nil
	}
	if !ok2 {
		return nil, fmt.Errorf("apiSrvMaxReplicas has to be specified for shooted seed API server autoscaler")
	}

	var apiServerAutoscaler ShootedSeedAPIServerAutoscaler

	if ok1 {
		minReplicas, err := parseInt32(minReplicasString)
		if err != nil {
			return nil, err
		}
		apiServerAutoscaler.MinReplicas = &minReplicas
	}

	maxReplicas, err := parseInt32(maxReplicasString)
	if err != nil {
		return nil, err
	}
	apiServerAutoscaler.MaxReplicas = maxReplicas

	return &apiServerAutoscaler, nil
}

func validateShootedSeed(shootedSeed *ShootedSeed, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if shootedSeed.APIServer != nil {
		allErrs = append(validateShootedSeedAPIServer(shootedSeed.APIServer, fldPath.Child("apiServer")))
	}

	return allErrs
}

func validateShootedSeedAPIServer(apiServer *ShootedSeedAPIServer, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if apiServer.Replicas != nil && *apiServer.Replicas < 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("replicas"), *apiServer.Replicas, "must be greater than 0"))
	}
	if apiServer.Autoscaler != nil {
		allErrs = append(allErrs, validateShootedSeedAPIServerAutoscaler(apiServer.Autoscaler, fldPath.Child("autoscaler"))...)
	}

	return allErrs
}

func validateShootedSeedAPIServerAutoscaler(autoscaler *ShootedSeedAPIServerAutoscaler, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if autoscaler.MinReplicas != nil && *autoscaler.MinReplicas < 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("minReplicas"), *autoscaler.MinReplicas, "must be greater than 0"))
	}
	if autoscaler.MaxReplicas < 1 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("maxReplicas"), autoscaler.MaxReplicas, "must be greater than 0"))
	}
	if autoscaler.MinReplicas != nil && autoscaler.MaxReplicas < *autoscaler.MinReplicas {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("maxReplicas"), autoscaler.MaxReplicas, "must be greater than or equal to `minReplicas`"))
	}

	return allErrs
}

func setDefaults_ShootedSeed(shootedSeed *ShootedSeed) {
	if shootedSeed.APIServer == nil {
		shootedSeed.APIServer = &ShootedSeedAPIServer{}
	}
	setDefaults_ShootedSeedAPIServer(shootedSeed.APIServer)
}

func setDefaults_ShootedSeedAPIServer(apiServer *ShootedSeedAPIServer) {
	if apiServer.Replicas == nil {
		three := int32(3)
		apiServer.Replicas = &three
	}
	if apiServer.Autoscaler == nil {
		apiServer.Autoscaler = &ShootedSeedAPIServerAutoscaler{
			MaxReplicas: 3,
		}
	}
	setDefaults_ShootedSeedAPIServerAutoscaler(apiServer.Autoscaler)
}

func minInt32(a int32, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func setDefaults_ShootedSeedAPIServerAutoscaler(autoscaler *ShootedSeedAPIServerAutoscaler) {
	if autoscaler.MinReplicas == nil {
		minReplicas := minInt32(3, autoscaler.MaxReplicas)
		autoscaler.MinReplicas = &minReplicas
	}
}

// ReadShootedSeed determines whether the Shoot has been marked to be registered automatically as a Seed cluster.
func ReadShootedSeed(shoot *gardencorev1beta1.Shoot) (*ShootedSeed, error) {
	if shoot.Namespace != v1beta1constants.GardenNamespace || shoot.Annotations == nil {
		return nil, nil
	}

	val, ok := shoot.Annotations[v1beta1constants.AnnotationShootUseAsSeed]
	if !ok {
		return nil, nil
	}

	shootedSeed, err := parseShootedSeed(val)
	if err != nil {
		return nil, err
	}

	if shootedSeed == nil {
		return nil, nil
	}

	setDefaults_ShootedSeed(shootedSeed)

	if errs := validateShootedSeed(shootedSeed, nil); len(errs) > 0 {
		return nil, errs.ToAggregate()
	}

	return shootedSeed, nil
}

// HibernationIsEnabled checks if the given shoot's desired state is hibernated.
func HibernationIsEnabled(shoot *gardencorev1beta1.Shoot) bool {
	return shoot.Spec.Hibernation != nil && shoot.Spec.Hibernation.Enabled != nil && *shoot.Spec.Hibernation.Enabled
}

// ShootWantsClusterAutoscaler checks if the given Shoot needs a cluster autoscaler.
// This is determined by checking whether one of the Shoot workers has a different
// Maximum than Minimum.
func ShootWantsClusterAutoscaler(shoot *gardencorev1beta1.Shoot) (bool, error) {
	for _, worker := range shoot.Spec.Provider.Workers {
		if worker.Maximum > worker.Minimum {
			return true, nil
		}
	}
	return false, nil
}

// ShootIgnoresAlerts checks if the alerts for the annotated shoot cluster should be ignored.
func ShootIgnoresAlerts(shoot *gardencorev1beta1.Shoot) bool {
	ignore := false
	if value, ok := shoot.Annotations[v1beta1constants.AnnotationShootIgnoreAlerts]; ok {
		ignore, _ = strconv.ParseBool(value)
	}
	return ignore
}

// ShootWantsBasicAuthentication returns true if basic authentication is not configured or
// if it is set explicitly to 'true'.
func ShootWantsBasicAuthentication(shoot *gardencorev1beta1.Shoot) bool {
	kubeAPIServerConfig := shoot.Spec.Kubernetes.KubeAPIServer
	if kubeAPIServerConfig == nil {
		return true
	}
	if kubeAPIServerConfig.EnableBasicAuthentication == nil {
		return true
	}
	return *kubeAPIServerConfig.EnableBasicAuthentication
}

// DetermineMachineImageForName finds the cloud specific machine images in the <cloudProfile> for the given <name> and
// region. In case it does not find the machine image with the <name>, it returns false. Otherwise, true and the
// cloud-specific machine image will be returned.
func DetermineMachineImageForName(cloudProfile *gardencorev1beta1.CloudProfile, name string) (bool, gardencorev1beta1.MachineImage, error) {
	for _, image := range cloudProfile.Spec.MachineImages {
		if strings.ToLower(image.Name) == strings.ToLower(name) {
			return true, image, nil
		}
	}
	return false, gardencorev1beta1.MachineImage{}, nil
}

// ShootMachineImageVersionExists checks if the shoot machine image (name, version) exists in the machine image constraint and returns true if yes and the index in the versions slice
func ShootMachineImageVersionExists(constraint gardencorev1beta1.MachineImage, image gardencorev1beta1.ShootMachineImage) (bool, int) {
	if constraint.Name != image.Name {
		return false, 0
	}

	for index, v := range constraint.Versions {
		if v.Version == image.Version {
			return true, index
		}
	}

	return false, 0
}

// DetermineLatestMachineImageVersion determines the latest MachineImageVersion from a MachineImage
func DetermineLatestMachineImageVersion(image gardencorev1beta1.MachineImage) (*semver.Version, gardencorev1beta1.ExpirableVersion, error) {
	var (
		latestSemVerVersion       *semver.Version
		latestMachineImageVersion gardencorev1beta1.ExpirableVersion
	)

	for _, imageVersion := range image.Versions {
		v, err := semver.NewVersion(imageVersion.Version)
		if err != nil {
			return nil, gardencorev1beta1.ExpirableVersion{}, fmt.Errorf("error while parsing machine image version '%s' of machine image '%s': version not valid: %s", imageVersion.Version, image.Name, err.Error())
		}
		if latestSemVerVersion == nil || v.GreaterThan(latestSemVerVersion) {
			latestSemVerVersion = v
			latestMachineImageVersion = imageVersion
		}
	}
	return latestSemVerVersion, latestMachineImageVersion, nil
}

// GetShootMachineImageFromLatestMachineImageVersion determines the latest version in a machine image and returns that as a ShootMachineImage
func GetShootMachineImageFromLatestMachineImageVersion(image gardencorev1beta1.MachineImage) (*semver.Version, gardencorev1beta1.ShootMachineImage, error) {
	latestSemVerVersion, latestImage, err := DetermineLatestMachineImageVersion(image)
	if err != nil {
		return nil, gardencorev1beta1.ShootMachineImage{}, err
	}
	return latestSemVerVersion, gardencorev1beta1.ShootMachineImage{Name: image.Name, Version: latestImage.Version}, nil
}

// UpdateMachineImages updates the machine images in place.
func UpdateMachineImages(workers []gardencorev1beta1.Worker, machineImages []*gardencorev1beta1.ShootMachineImage) {
	for _, machineImage := range machineImages {
		for idx, worker := range workers {
			if worker.Machine.Image != nil && machineImage.Name == worker.Machine.Image.Name {
				logger.Logger.Infof("Updating worker images of worker '%s' from version %s to version %s", worker.Name, worker.Machine.Image.Version, machineImage.Version)
				workers[idx].Machine.Image = machineImage
			}
		}
	}
}

// KubernetesVersionExistsInCloudProfile checks if the given Kubernetes version exists in the CloudProfile
func KubernetesVersionExistsInCloudProfile(cloudProfile *gardencorev1beta1.CloudProfile, currentVersion string) (bool, gardencorev1beta1.ExpirableVersion, error) {
	for _, version := range cloudProfile.Spec.Kubernetes.Versions {
		ok, err := utils.CompareVersions(version.Version, "=", currentVersion)
		if err != nil {
			return false, gardencorev1beta1.ExpirableVersion{}, err
		}
		if ok {
			return true, version, nil
		}
	}
	return false, gardencorev1beta1.ExpirableVersion{}, nil
}

// DetermineLatestKubernetesPatchVersion finds the latest Kubernetes patch version in the <cloudProfile> compared
// to the given <currentVersion>. In case it does not find a newer patch version, it returns false. Otherwise,
// true and the found version will be returned.
func DetermineLatestKubernetesPatchVersion(cloudProfile *gardencorev1beta1.CloudProfile, currentVersion string) (bool, string, error) {
	ok, newerVersions, _, err := determineNextKubernetesVersions(cloudProfile, currentVersion, "~")
	if err != nil || !ok {
		return ok, "", err
	}
	sort.Strings(newerVersions)
	return true, newerVersions[len(newerVersions)-1], nil
}

// DetermineNextKubernetesMinorVersion finds the next available Kubernetes minor version in the <cloudProfile> compared
// to the given <currentVersion>. In case it does not find a newer minor version, it returns false. Otherwise,
// true and the found version will be returned.
func DetermineNextKubernetesMinorVersion(cloudProfile *gardencorev1beta1.CloudProfile, currentVersion string) (bool, string, error) {
	ok, newerVersions, _, err := determineNextKubernetesVersions(cloudProfile, currentVersion, "^")
	if err != nil || !ok {
		return ok, "", err
	}
	sort.Strings(newerVersions)
	return true, newerVersions[0], nil
}

// determineKubernetesVersions finds newer Kubernetes versions in the <cloudProfile> compared
// with the <operator> to the given <currentVersion>. The <operator> has to be a github.com/Masterminds/semver
// range comparison symbol. In case it does not find a newer version, it returns false. Otherwise,
// true and the found version will be returned.
func determineNextKubernetesVersions(cloudProfile *gardencorev1beta1.CloudProfile, currentVersion, operator string) (bool, []string, []gardencorev1beta1.ExpirableVersion, error) {
	var (
		newerVersions       = []gardencorev1beta1.ExpirableVersion{}
		newerVersionsString = []string{}
	)

	for _, version := range cloudProfile.Spec.Kubernetes.Versions {
		ok, err := utils.CompareVersions(version.Version, operator, currentVersion)
		if err != nil {
			return false, []string{}, []gardencorev1beta1.ExpirableVersion{}, err
		}
		if version.Version != currentVersion && ok {
			newerVersions = append(newerVersions, version)
			newerVersionsString = append(newerVersionsString, version.Version)
		}
	}

	if len(newerVersions) == 0 {
		return false, []string{}, []gardencorev1beta1.ExpirableVersion{}, nil
	}

	return true, newerVersionsString, newerVersions, nil
}

// SetMachineImageVersionsToMachineImage sets imageVersions to the matching imageName in the machineImages.
func SetMachineImageVersionsToMachineImage(machineImages []gardencorev1beta1.MachineImage, imageName string, imageVersions []gardencorev1beta1.ExpirableVersion) ([]gardencorev1beta1.MachineImage, error) {
	for index, image := range machineImages {
		if strings.ToLower(image.Name) == strings.ToLower(imageName) {
			machineImages[index].Versions = imageVersions
			return machineImages, nil
		}
	}
	return nil, fmt.Errorf("machine image with name '%s' could not be found", imageName)
}

// GetDefaultMachineImageFromCloudProfile gets the first MachineImage from the CloudProfile
func GetDefaultMachineImageFromCloudProfile(profile gardencorev1beta1.CloudProfile) *gardencorev1beta1.MachineImage {
	if len(profile.Spec.MachineImages) == 0 {
		return nil
	}
	return &profile.Spec.MachineImages[0]
}
