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

// SecretBindingInformer provides access to a shared informer and lister for
// SecretBindings.
type SecretBindingInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1beta1.SecretBindingLister
}

type secretBindingInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewSecretBindingInformer constructs a new informer for SecretBinding type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewSecretBindingInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredSecretBindingInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredSecretBindingInformer constructs a new informer for SecretBinding type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredSecretBindingInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CoreV1beta1().SecretBindings(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CoreV1beta1().SecretBindings(namespace).Watch(options)
			},
		},
		&corev1beta1.SecretBinding{},
		resyncPeriod,
		indexers,
	)
}

func (f *secretBindingInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredSecretBindingInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *secretBindingInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&corev1beta1.SecretBinding{}, f.defaultInformer)
}

func (f *secretBindingInformer) Lister() v1beta1.SecretBindingLister {
	return v1beta1.NewSecretBindingLister(f.Informer().GetIndexer())
}
