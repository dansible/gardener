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

package common

import (
	v1alpha1constants "github.com/gardener/gardener/pkg/apis/core/v1alpha1/constants"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// BackupBucketName is a constant for the name of bucket of object storage.
	BackupBucketName = "bucketName"

	// BackupSecretName defines the name of the secret containing the credentials which are required to
	// authenticate against the respective cloud provider (required to store the backups of Shoot clusters).
	BackupSecretName = "etcd-backup"

	// BasicAuthSecretName is the name of the secret containing basic authentication credentials for the kube-apiserver.
	BasicAuthSecretName = "kube-apiserver-basic-auth"

	// ChartPath is the path to the Helm charts.
	ChartPath = "charts"

	// CloudConfigPrefix is a constant for the prefix which is added to secret storing the original cloud config (which
	// is being downloaded from the cloud-config-downloader process)
	CloudConfigPrefix = "cloud-config"

	// CloudConfigFilePath is the path on the shoot worker nodes to which the operating system specific configuration
	// will be downloaded.
	CloudConfigFilePath = "/var/lib/cloud-config-downloader/downloads/cloud_config"

	// CloudProviderConfigName is the name of the configmap containing the cloud provider config.
	CloudProviderConfigName = "cloud-provider-config"

	// CloudProviderConfigMapKey is the key storing the cloud provider config as value in the cloud provider configmap.
	CloudProviderConfigMapKey = "cloudprovider.conf"

	// CloudPurposeShoot is a constant used while instantiating a cloud botanist for the Shoot cluster.
	CloudPurposeShoot = "shoot"

	// CloudPurposeSeed is a constant used while instantiating a cloud botanist for the Seed cluster.
	CloudPurposeSeed = "seed"

	// ConfirmationDeletion is an annotation on a Shoot resource whose value must be set to "true" in order to
	// allow deleting the Shoot (if the annotation is not set any DELETE request will be denied).
	ConfirmationDeletion = "confirmation.garden.sapcloud.io/deletion"

	// ControllerManagerInternalConfigMapName is the name of the internal config map in which the Gardener controller
	// manager stores its configuration.
	ControllerManagerInternalConfigMapName = "gardener-controller-manager-internal-config"

	// DNSProviderDeprecated is the key for an annotation on a Kubernetes Secret object whose value must point to a valid
	// DNS provider.
	// deprecated
	DNSProviderDeprecated = "dns.garden.sapcloud.io/provider"

	// DNSDomainDeprecated is the key for an annotation on a Kubernetes Secret object whose value must point to a valid
	// domain name.
	// deprecated
	DNSDomainDeprecated = "dns.garden.sapcloud.io/domain"

	// DNSProvider is the key for an annotation on a Kubernetes Secret object whose value must point to a valid
	// DNS provider.
	DNSProvider = "dns.gardener.cloud/provider"

	// DNSDomain is the key for an annotation on a Kubernetes Secret object whose value must point to a valid
	// domain name.
	DNSDomain = "dns.gardener.cloud/domain"

	// DNSIncludeZones is the key for an annotation on a Kubernetes Secret object whose value must point to a list
	// of zones that shall be included.
	DNSIncludeZones = "dns.gardener.cloud/include-zones"

	// DNSExcludeZones is the key for an annotation on a Kubernetes Secret object whose value must point to a list
	// of zones that shall be excluded.
	DNSExcludeZones = "dns.gardener.cloud/exclude-zones"

	// EtcdRoleMain is the constant defining the role for main etcd storing data about objects in Shoot.
	EtcdRoleMain = "main"

	// EtcdRoleEvents is the constant defining the role for etcd storing events in Shoot.
	EtcdRoleEvents = "events"

	// EtcdEncryptionSecretName is the name of the shoot-specific secret which contains
	// that shoot's EncryptionConfiguration. The EncryptionConfiguration contains a key
	// which the shoot's apiserver uses for encrypting selected etcd content.
	// Should match charts/seed-controlplane/charts/kube-apiserver/templates/kube-apiserver.yaml
	EtcdEncryptionSecretName = "etcd-encryption-secret"

	// EtcdEncryptionSecretFileName is the name of the file within the EncryptionConfiguration
	// which is made available as volume mount to the shoot's apiserver.
	// Should match charts/seed-controlplane/charts/kube-apiserver/templates/kube-apiserver.yaml
	EtcdEncryptionSecretFileName = "encryption-configuration.yaml"

	// EtcdEncryptionChecksumAnnotationName is the name of the annotation with which to annotate
	// the EncryptionConfiguration secret to denote the checksum of the EncryptionConfiguration
	// that was used when last rewriting secrets.
	EtcdEncryptionChecksumAnnotationName = "shoot.gardener.cloud/etcd-encryption-configuration-checksum"

	// EtcdEncryptionChecksumLabelName is the name of the label which is added to the shoot
	// secrets after rewriting them to ensure that successfully rewritten secrets are not
	// (unnecessarily) rewritten during each reconciliation.
	EtcdEncryptionChecksumLabelName = "shoot.gardener.cloud/etcd-encryption-configuration-checksum"

	// EtcdEncryptionForcePlaintextAnnotationName is the name of the annotation with which to annotate
	// the EncryptionConfiguration secret to force the decryption of shoot secrets
	EtcdEncryptionForcePlaintextAnnotationName = "shoot.gardener.cloud/etcd-encryption-force-plaintext-secrets"

	// EtcdEncryptionEncryptedResourceSecrets is the name of the secret resource to be encrypted
	EtcdEncryptionEncryptedResourceSecrets = "secrets"

	// EtcdEncryptionKeyPrefix is the prefix for the key name of the EncryptionConfiguration's key
	EtcdEncryptionKeyPrefix = "key"

	// EtcdEncryptionKeySecretLen is the expected length in bytes of the EncryptionConfiguration's key
	EtcdEncryptionKeySecretLen = 32

	// GardenRoleDefaultDomain is the value of the GardenRole key indicating type 'default-domain'.
	GardenRoleDefaultDomain = "default-domain"

	// GardenRoleInternalDomain is the value of the GardenRole key indicating type 'internal-domain'.
	GardenRoleInternalDomain = "internal-domain"

	// GardenRoleAlertingSMTP is the value of the GardenRole key indicating type 'alerting-smtp'.
	GardenRoleAlertingSMTP = "alerting-smtp"

	// GardenRoleOpenVPNDiffieHellman is the value of the GardenRole key indicating type 'openvpn-diffie-hellman'.
	GardenRoleOpenVPNDiffieHellman = "openvpn-diffie-hellman"

	// GardenRoleMembers is the value of GardenRole key indicating type 'members'.
	GardenRoleMembers = "members"

	// GardenRoleGlobalMonitoring is the value of the GardenRole key indicating type 'global-monitoring'
	GardenRoleGlobalMonitoring = "global-monitoring"

	// GardenRoleAlerting is the value of GardenRole key indicating type 'alerting'.
	GardenRoleAlerting = "alerting"

	// GardenRoleHvpa is the value of GardenRole key indicating type 'hvpa'.
	GardenRoleHvpa = "hvpa"

	// GardenCreatedBy is the key for an annotation of a Shoot cluster whose value indicates contains the username
	// of the user that created the resource.
	GardenCreatedBy = "garden.sapcloud.io/createdBy"

	// GrafanaOperatorsPrefix is a constant for a prefix used for the operators Grafana instance.
	GrafanaOperatorsPrefix = "go"

	// GrafanaUsersPrefix is a constant for a prefix used for the users Grafana instance.
	GrafanaUsersPrefix = "gu"

	// IngressPrefix is the part of a FQDN which will be used to construct the domain name for an ingress controller of
	// a Shoot cluster. For example, when a Shoot specifies domain 'cluster.example.com', the ingress domain would be
	// '*.<IngressPrefix>.cluster.example.com'.
	IngressPrefix = "ingress"

	// APIServerPrefix is the part of a FQDN which will be used to construct the domain name for the kube-apiserver of
	// a Shoot cluster. For example, when a Shoot specifies domain 'cluster.example.com', the apiserver domain would be
	// 'api.cluster.example.com'.
	APIServerPrefix = "api"

	// InternalDomainKey is a key which must be present in an internal domain constructed for a Shoot cluster. If the
	// configured internal domain already contains it, it won't be added twice. If it does not contain it, it will be
	// appended.
	InternalDomainKey = "internal"

	// KubeControllerManagerServerName is the name of the kube-controller-manager server.
	KubeControllerManagerServerName = "kube-controller-manager-server"

	// MachineControllerManagerDeploymentName is the name of the machine-controller-manager deployment.
	MachineControllerManagerDeploymentName = "machine-controller-manager"

	// KubeSchedulerServerName is the name of the kube-scheduler server.
	KubeSchedulerServerName = "kube-scheduler-server"

	// CalicoKubeControllersDeploymentName is the name of calico-kube-controllers deployment.
	CalicoKubeControllersDeploymentName = "calico-kube-controllers"

	// CoreDNSDeploymentName is the name of the coredns deployment.
	CoreDNSDeploymentName = "coredns"

	// VPNShootDeploymentName is the name of the vpn-shoot deployment.
	VPNShootDeploymentName = "vpn-shoot"

	// MetricsServerDeploymentName is the name of the metrics-server deployment.
	MetricsServerDeploymentName = "metrics-server"

	// CalicoNodeDaemonSetName is the name of the calico-node daemon set.
	CalicoNodeDaemonSetName = "calico-node"

	// KubeProxyDaemonSetName is the name of the kube-proxy daemon set.
	KubeProxyDaemonSetName = "kube-proxy"

	// NodeProblemDetectorDaemonSetName is the name of the node-problem-detector daemon set.
	NodeProblemDetectorDaemonSetName = "node-problem-detector"

	// NodeExporterDaemonSetName is the name of the node-exporter daemon set.
	NodeExporterDaemonSetName = "node-exporter"

	// KibanaAdminIngressCredentialsSecretName is the name of the secret which holds admin credentials.
	KibanaAdminIngressCredentialsSecretName = "logging-ingress-credentials"

	// KubecfgUsername is the username for the token used for the kubeconfig the shoot.
	KubecfgUsername = "system:cluster-admin"

	// KubecfgSecretName is the name of the kubecfg secret.
	KubecfgSecretName = "kubecfg"

	// KubecfgInternalSecretName is the name of the kubecfg secret with cluster IP access.
	KubecfgInternalSecretName = "kubecfg-internal"

	// KubeAPIServerHealthCheck is a key for the kube-apiserver-health-check user.
	KubeAPIServerHealthCheck = "kube-apiserver-health-check"

	// StaticTokenSecretName is the name of the secret containing static tokens for the kube-apiserver.
	StaticTokenSecretName = "static-token"

	// FluentBitDaemonSetName is the name of the fluent-bit daemon set.
	FluentBitDaemonSetName = "fluent-bit"

	// FluentdEsStatefulSetName is the name of the fluentd-es stateful set.
	FluentdEsStatefulSetName = "fluentd-es"

	// ProjectPrefix is the prefix of namespaces representing projects.
	ProjectPrefix = "garden-"

	// ProjectName is they key of a label on namespaces whose value holds the project name. Usually, the label is set
	// by the Gardener Dashboard.
	ProjectName = "project.garden.sapcloud.io/name"

	// NamespaceProject is they key of a label on namespace whose value holds the project uid.
	NamespaceProject = "namespace.garden.sapcloud.io/project"

	// SecretRefChecksumAnnotation is the annotation key for checksum of referred secret in resource spec.
	SecretRefChecksumAnnotation = "checksum/secret.data"

	// ShootExpirationTimestamp is an annotation on a Shoot resource whose value represents the time when the Shoot lifetime
	// is expired. The lifetime can be extended, but at most by the minimal value of the 'clusterLifetimeDays' property
	// of referenced quotas.
	ShootExpirationTimestamp = "shoot.garden.sapcloud.io/expirationTimestamp"

	// ShootNoCleanup is a constant for a label on a resource indicating the the Gardener cleaner should not delete this
	// resource when cleaning a shoot during the deletion flow.
	ShootNoCleanup = "shoot.gardener.cloud/no-cleanup"

	// ShootStatus is a constant for a label on a Shoot resource indicating that the Shoot's health.
	// Shoot Care controller and can be used to easily identify Shoot clusters with certain states.
	ShootStatus = "shoot.garden.sapcloud.io/status"

	// ShootUnhealthy is a constant for a label on a Shoot resource indicating that the Shoot is unhealthy. It is set and unset by the
	// Shoot Care controller and can be used to easily identify Shoot clusters with issues.
	// Deprecated: Use ShootStatus instead
	ShootUnhealthy = "shoot.garden.sapcloud.io/unhealthy"

	// ShootOperation is a constant for an annotation on a Shoot in a failed state indicating that an operation shall be performed.
	ShootOperation = "shoot.garden.sapcloud.io/operation"

	// ShootOperationMaintain is a constant for an annotation on a Shoot indicating that the Shoot maintenance shall be executed as soon as
	// possible.
	ShootOperationMaintain = "maintain"

	// ShootOperationRotateKubeconfigCredentials is a constant for an annotation on a Shoot indicating that the credentials contained in the
	// kubeconfig that is handed out to the user shall be rotated.
	ShootOperationRotateKubeconfigCredentials = "rotate-kubeconfig-credentials"

	// ShootTasks is a constant for an annotation on a Shoot which states that certain tasks should be done.
	ShootTasks = "shoot.garden.sapcloud.io/tasks"

	// ShootTaskDeployInfrastructure is a name for a Shoot's infrastructure deployment task.
	ShootTaskDeployInfrastructure = "deployInfrastructure"

	// ShootOperationRetry is a constant for an annotation on a Shoot indicating that a failed Shoot reconciliation shall be retried.
	ShootOperationRetry = "retry"

	// ShootOperationReconcile is a constant for an annotation on a Shoot indicating that a Shoot reconciliation shall be triggered.
	ShootOperationReconcile = "reconcile"

	// ShootSyncPeriod is a constant for an annotation on a Shoot which may be used to overwrite the global Shoot controller sync period.
	// The value must be a duration. It can also be used to disable the reconciliation at all by setting it to 0m. Disabling the reconciliation
	// does only mean that the period reconciliation is disabled. However, when the Gardener is restarted/redeployed or the specification is
	// changed then the reconciliation flow will be executed.
	ShootSyncPeriod = "shoot.garden.sapcloud.io/sync-period"

	// ShootIgnore is a constant for an annotation on a Shoot which may be used to tell the Gardener that the Shoot with this name should be
	// ignored completely. That means that the Shoot will never reach the reconciliation flow (independent of the operation (create/update/
	// delete)).
	ShootIgnore = "shoot.garden.sapcloud.io/ignore"

	// AnnotatePersistentVolumeMinimumSize is used to specify the minimum size of persistent volume in the cluster
	AnnotatePersistentVolumeMinimumSize = "persistentvolume.garden.sapcloud.io/minimumSize"

	// AnnotatePersistentVolumeProvider is used to tell volume provider in the k8s cluster
	AnnotatePersistentVolumeProvider = "persistentvolume.garden.sapcloud.io/provider"

	// BackupNamespacePrefix is a constant for backup namespace created for shoot's backup infrastructure related resources.
	BackupNamespacePrefix = "backup"

	// GardenerResourceManagerImageName is the name of the GardenerResourceManager image.
	GardenerResourceManagerImageName = "gardener-resource-manager"

	// CoreDNSImageName is the name of the CoreDNS image.
	CoreDNSImageName = "coredns"

	// NodeProblemDetectorImageName is the name of the node-problem-detector image.
	NodeProblemDetectorImageName = "node-problem-detector"

	// HyperkubeImageName is the name of the Hyperkube image.
	HyperkubeImageName = "hyperkube"

	// MetricsServerImageName is the name of the MetricsServer image.
	MetricsServerImageName = "metrics-server"

	// VPNShootImageName is the name of the VPNShoot image.
	VPNShootImageName = "vpn-shoot"

	// VPNSeedImageName is the name of the VPNSeed image.
	VPNSeedImageName = "vpn-seed"

	// NodeExporterImageName is the name of the NodeExporter image.
	NodeExporterImageName = "node-exporter"

	// KubernetesDashboardImageName is the name of the KubernetesDashboard image.
	KubernetesDashboardImageName = "kubernetes-dashboard"

	// BusyboxImageName is the name of the Busybox image.
	BusyboxImageName = "busybox"

	// NginxIngressControllerImageName is the name of the NginxIngressController image.
	NginxIngressControllerImageName = "nginx-ingress-controller"

	// IngressDefaultBackendImageName is the name of the IngressDefaultBackend image.
	IngressDefaultBackendImageName = "ingress-default-backend"

	// ClusterAutoscalerImageName is the name of the ClusterAutoscaler image.
	ClusterAutoscalerImageName = "cluster-autoscaler"

	// AlertManagerImageName is the name of the AlertManager image.
	AlertManagerImageName = "alertmanager"

	// ConfigMapReloaderImageName is the name of the ConfigMapReloader image.
	ConfigMapReloaderImageName = "configmap-reloader"

	// GrafanaImageName is the name of the Grafana image.
	GrafanaImageName = "grafana"

	// PrometheusImageName is the name of the Prometheus image.
	PrometheusImageName = "prometheus"

	// BlackboxExporterImageName is the name of the BlackboxExporter image.
	BlackboxExporterImageName = "blackbox-exporter"

	// KubeStateMetricsImageName is the name of the KubeStateMetrics image.
	KubeStateMetricsImageName = "kube-state-metrics"

	// ETCDImageName is the name of the ETCD image.
	ETCDImageName = "etcd"

	// CSINodeDriverRegistrarImageName is the name of driver registrar - https://github.com/kubernetes-csi/node-driver-registrar
	CSINodeDriverRegistrarImageName = "csi-node-driver-registrar"

	// CSIPluginAlicloudImageName is the name of csi plugin for Alicloud - https://github.com/AliyunContainerService/csi-plugin
	CSIPluginAlicloudImageName = "csi-plugin-alicloud"

	// CSIPluginPacketImageName is the name of csi plugin for Packet - https://github.com/packethost/csi-packet
	CSIPluginPacketImageName = "packet-storage-interface"

	// PauseContainerImageName is the name of the PauseContainer image.
	PauseContainerImageName = "pause-container"

	// ElasticsearchImageName is the name of the Elastic-Search image used for logging
	ElasticsearchImageName = "elasticsearch-oss"

	// ElasticsearchMetricsExporterImageName is the name of the metrics exporter image used to fetch elasticsearch metrics.
	ElasticsearchMetricsExporterImageName = "elasticsearch-metrics-exporter"

	// ElasticsearchSearchguardImageName is the name of the Elastic-Search image with installed searchguard plugin used for logging
	ElasticsearchSearchguardImageName = "elasticsearch-searchguard-oss"

	// CuratorImageName is the name of the curator image used to alter the Elastic-search logs
	CuratorImageName = "curator-es"

	// KibanaImageName is the name of the Kibana image used for logging  UI
	KibanaImageName = "kibana-oss"

	// SearchguardImageName is the name of the Searchguard image used for updating the users and roles
	SearchguardImageName = "sg-sgadmin"

	// FluentdEsImageName is the image of the Fluentd image used for logging
	FluentdEsImageName = "fluentd-es"

	// FluentBitImageName is the image of Fluent-bit image
	FluentBitImageName = "fluent-bit"

	// AlpineImageName is the name of alpine image
	AlpineImageName = "alpine"

	// AlpineIptablesImageName is the name of the alpine image with pre-installed iptable rules
	AlpineIptablesImageName = "alpine-iptables"

	// SeedSpecHash is a constant for a label on `ControllerInstallation`s (similar to `pod-template-hash` on `Pod`s).
	SeedSpecHash = "seed-spec-hash"

	// RegistrationSpecHash is a constant for a label on `ControllerInstallation`s (similar to `pod-template-hash` on `Pod`s).
	RegistrationSpecHash = "registration-spec-hash"

	// VpaAdmissionControllerImageName is the name of the vpa-admission-controller image
	VpaAdmissionControllerImageName = "vpa-admission-controller"

	// VpaRecommenderImageName is the name of the vpa-recommender image
	VpaRecommenderImageName = "vpa-recommender"

	// VpaUpdaterImageName is the name of the vpa-updater image
	VpaUpdaterImageName = "vpa-updater"

	// VpaExporterImageName is the name of the vpa-exporter image
	VpaExporterImageName = "vpa-exporter"

	// HvpaControllerImageName is the name of the hvpa-controller image
	HvpaControllerImageName = "hvpa-controller"

	// DependencyWatchdogImageName is the name of the dependency-watchdog image
	DependencyWatchdogImageName = "dependency-watchdog"

	// ServiceAccountSigningKeySecretDataKey is the data key of a signing key Kubernetes secret.
	ServiceAccountSigningKeySecretDataKey = "signing-key"
)

var (
	// RequiredControlPlaneDeployments is a set of the required shoot control plane deployments
	// running in the seed.
	RequiredControlPlaneDeployments = sets.NewString(
		v1alpha1constants.DeploymentNameGardenerResourceManager,
		v1alpha1constants.DeploymentNameKubeAPIServer,
		v1alpha1constants.DeploymentNameKubeControllerManager,
		v1alpha1constants.DeploymentNameKubeScheduler,
		MachineControllerManagerDeploymentName,
	)

	// RequiredControlPlaneStatefulSets is a set of the required shoot control plane stateful
	// sets running in the seed.
	RequiredControlPlaneStatefulSets = sets.NewString(
		v1alpha1constants.StatefulSetNameETCDMain,
		v1alpha1constants.StatefulSetNameETCDEvents,
	)

	// RequiredSystemComponentDeployments is a set of the required system components.
	RequiredSystemComponentDeployments = sets.NewString(
		CalicoKubeControllersDeploymentName,
		CoreDNSDeploymentName,
		VPNShootDeploymentName,
		MetricsServerDeploymentName,
	)

	// RequiredSystemComponentDaemonSets is a set of the required shoot control plane daemon sets.
	RequiredSystemComponentDaemonSets = sets.NewString(
		CalicoNodeDaemonSetName,
		KubeProxyDaemonSetName,
		NodeProblemDetectorDaemonSetName,
	)

	// RequiredMonitoringSeedDeployments is a set of the required seed monitoring deployments.
	RequiredMonitoringSeedDeployments = sets.NewString(
		v1alpha1constants.DeploymentNameGrafanaOperators,
		v1alpha1constants.DeploymentNameGrafanaUsers,
		v1alpha1constants.DeploymentNameKubeStateMetricsSeed,
		v1alpha1constants.DeploymentNameKubeStateMetricsShoot,
	)

	// RequiredMonitoringShootDaemonSets is a set of the required shoot monitoring daemon sets.
	RequiredMonitoringShootDaemonSets = sets.NewString(
		NodeExporterDaemonSetName,
	)

	// RequiredLoggingStatefulSets is a set of the required logging stateful sets.
	RequiredLoggingStatefulSets = sets.NewString(
		v1alpha1constants.StatefulSetNameElasticSearch,
	)

	// RequiredLoggingDeployments is a set of the required logging deployments.
	RequiredLoggingDeployments = sets.NewString(
		v1alpha1constants.DeploymentNameKibana,
	)
)
