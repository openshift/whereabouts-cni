/*
Copyright 2025 The Kubernetes Authors

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

package v1alpha1

import (
	v1alpha1 "github.com/k8snetworkplumbingwg/whereabouts/pkg/api/whereabouts.cni.cncf.io/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
	"k8s.io/client-go/tools/cache"
)

// IPPoolLister helps list IPPools.
// All objects returned here must be treated as read-only.
type IPPoolLister interface {
	// List lists all IPPools in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.IPPool, err error)
	// IPPools returns an object that can list and get IPPools.
	IPPools(namespace string) IPPoolNamespaceLister
	IPPoolListerExpansion
}

// iPPoolLister implements the IPPoolLister interface.
type iPPoolLister struct {
	listers.ResourceIndexer[*v1alpha1.IPPool]
}

// NewIPPoolLister returns a new IPPoolLister.
func NewIPPoolLister(indexer cache.Indexer) IPPoolLister {
	return &iPPoolLister{listers.New[*v1alpha1.IPPool](indexer, v1alpha1.Resource("ippool"))}
}

// IPPools returns an object that can list and get IPPools.
func (s *iPPoolLister) IPPools(namespace string) IPPoolNamespaceLister {
	return iPPoolNamespaceLister{listers.NewNamespaced[*v1alpha1.IPPool](s.ResourceIndexer, namespace)}
}

// IPPoolNamespaceLister helps list and get IPPools.
// All objects returned here must be treated as read-only.
type IPPoolNamespaceLister interface {
	// List lists all IPPools in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.IPPool, err error)
	// Get retrieves the IPPool from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.IPPool, error)
	IPPoolNamespaceListerExpansion
}

// iPPoolNamespaceLister implements the IPPoolNamespaceLister
// interface.
type iPPoolNamespaceLister struct {
	listers.ResourceIndexer[*v1alpha1.IPPool]
}
