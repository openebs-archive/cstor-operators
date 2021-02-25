/*
Copyright 2021 The OpenEBS Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/openebs/api/v2/pkg/apis/cstor/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// CStorRestoreLister helps list CStorRestores.
// All objects returned here must be treated as read-only.
type CStorRestoreLister interface {
	// List lists all CStorRestores in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.CStorRestore, err error)
	// CStorRestores returns an object that can list and get CStorRestores.
	CStorRestores(namespace string) CStorRestoreNamespaceLister
	CStorRestoreListerExpansion
}

// cStorRestoreLister implements the CStorRestoreLister interface.
type cStorRestoreLister struct {
	indexer cache.Indexer
}

// NewCStorRestoreLister returns a new CStorRestoreLister.
func NewCStorRestoreLister(indexer cache.Indexer) CStorRestoreLister {
	return &cStorRestoreLister{indexer: indexer}
}

// List lists all CStorRestores in the indexer.
func (s *cStorRestoreLister) List(selector labels.Selector) (ret []*v1.CStorRestore, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.CStorRestore))
	})
	return ret, err
}

// CStorRestores returns an object that can list and get CStorRestores.
func (s *cStorRestoreLister) CStorRestores(namespace string) CStorRestoreNamespaceLister {
	return cStorRestoreNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// CStorRestoreNamespaceLister helps list and get CStorRestores.
// All objects returned here must be treated as read-only.
type CStorRestoreNamespaceLister interface {
	// List lists all CStorRestores in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.CStorRestore, err error)
	// Get retrieves the CStorRestore from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.CStorRestore, error)
	CStorRestoreNamespaceListerExpansion
}

// cStorRestoreNamespaceLister implements the CStorRestoreNamespaceLister
// interface.
type cStorRestoreNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all CStorRestores in the indexer for a given namespace.
func (s cStorRestoreNamespaceLister) List(selector labels.Selector) (ret []*v1.CStorRestore, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.CStorRestore))
	})
	return ret, err
}

// Get retrieves the CStorRestore from the indexer for a given namespace and name.
func (s cStorRestoreNamespaceLister) Get(name string) (*v1.CStorRestore, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("cstorrestore"), name)
	}
	return obj.(*v1.CStorRestore), nil
}
