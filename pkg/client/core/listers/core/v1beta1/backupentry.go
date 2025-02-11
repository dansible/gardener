// Code generated by lister-gen. DO NOT EDIT.

package v1beta1

import (
	v1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// BackupEntryLister helps list BackupEntries.
type BackupEntryLister interface {
	// List lists all BackupEntries in the indexer.
	List(selector labels.Selector) (ret []*v1beta1.BackupEntry, err error)
	// BackupEntries returns an object that can list and get BackupEntries.
	BackupEntries(namespace string) BackupEntryNamespaceLister
	BackupEntryListerExpansion
}

// backupEntryLister implements the BackupEntryLister interface.
type backupEntryLister struct {
	indexer cache.Indexer
}

// NewBackupEntryLister returns a new BackupEntryLister.
func NewBackupEntryLister(indexer cache.Indexer) BackupEntryLister {
	return &backupEntryLister{indexer: indexer}
}

// List lists all BackupEntries in the indexer.
func (s *backupEntryLister) List(selector labels.Selector) (ret []*v1beta1.BackupEntry, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1beta1.BackupEntry))
	})
	return ret, err
}

// BackupEntries returns an object that can list and get BackupEntries.
func (s *backupEntryLister) BackupEntries(namespace string) BackupEntryNamespaceLister {
	return backupEntryNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// BackupEntryNamespaceLister helps list and get BackupEntries.
type BackupEntryNamespaceLister interface {
	// List lists all BackupEntries in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1beta1.BackupEntry, err error)
	// Get retrieves the BackupEntry from the indexer for a given namespace and name.
	Get(name string) (*v1beta1.BackupEntry, error)
	BackupEntryNamespaceListerExpansion
}

// backupEntryNamespaceLister implements the BackupEntryNamespaceLister
// interface.
type backupEntryNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all BackupEntries in the indexer for a given namespace.
func (s backupEntryNamespaceLister) List(selector labels.Selector) (ret []*v1beta1.BackupEntry, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1beta1.BackupEntry))
	})
	return ret, err
}

// Get retrieves the BackupEntry from the indexer for a given namespace and name.
func (s backupEntryNamespaceLister) Get(name string) (*v1beta1.BackupEntry, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1beta1.Resource("backupentry"), name)
	}
	return obj.(*v1beta1.BackupEntry), nil
}
