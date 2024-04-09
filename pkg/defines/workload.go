package defines

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Workload(chain unstructured.Unstructured) GroupVersionResourceKind {
	defines, _, err := unstructured.NestedMap(chain.UnstructuredContent(), "spec", "defines")
	if err != nil {
		panic(err)
	}
	gvr := GroupVersionResourceKind{
		Group:    defines["group"].(string),
		Version:  defines["version"].(string),
		Resource: strings.ToLower(defines["kind"].(string) + "s"),
		Kind:     defines["kind"].(string),
	}
	return gvr
}

type GroupVersionResourceKind struct {
	Group    string
	Version  string
	Resource string
	Kind     string
}

func (g *GroupVersionResourceKind) ToGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    g.Group,
		Version:  g.Version,
		Resource: g.Resource,
	}
}

func (g *GroupVersionResourceKind) ToGroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   g.Group,
		Version: g.Version,
		Kind:    g.Kind,
	}
}
