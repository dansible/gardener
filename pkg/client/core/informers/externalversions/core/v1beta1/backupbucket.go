// Code generated by informer-gen. DO NOT EDIT.

package v1beta1

import (
	time "time"

	corev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	versioned "github.com/gardener/gardener/pkg/client/core/clientset/versioned"
	internalinterfaces "github.com/gardener/gardener/pkg/client/core/informers/externalversions/internalinterfaces"
	v1beta1 "github.com/gardener/gardener/pkg/client/core/listers/core/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// BackupBucketInformer provides access to a shared informer and lister for
// BackupBuckets.
type BackupBucketInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1beta1.BackupBucketLister
}

type backupBucketInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewBackupBucketInformer constructs a new informer for BackupBucket type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewBackupBucketInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredBackupBucketInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredBackupBucketInformer constructs a new informer for BackupBucket type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredBackupBucketInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CoreV1beta1().BackupBuckets().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CoreV1beta1().BackupBuckets().Watch(options)
			},
		},
		&corev1beta1.BackupBucket{},
		resyncPeriod,
		indexers,
	)
}

func (f *backupBucketInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredBackupBucketInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *backupBucketInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&corev1beta1.BackupBucket{}, f.defaultInformer)
}

func (f *backupBucketInformer) Lister() v1beta1.BackupBucketLister {
	return v1beta1.NewBackupBucketLister(f.Informer().GetIndexer())
}
