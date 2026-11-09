package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

func (c *client) ApplyManifest(ctx context.Context, manifest []byte) error {
	if c.restCfg == nil {
		return fmt.Errorf("ApplyManifest requires a REST config (not available with fake client)")
	}

	dynClient, err := dynamic.NewForConfig(c.restCfg)
	if err != nil {
		return fmt.Errorf("create dynamic client: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(c.restCfg)
	if err != nil {
		return fmt.Errorf("create discovery client: %w", err)
	}

	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return fmt.Errorf("discover API resources: %w", err)
	}
	mapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(manifest), 4096)
	decSerializer := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	for {
		var rawObj unstructured.Unstructured
		if err := decoder.Decode(&rawObj); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode manifest document: %w", err)
		}

		if rawObj.Object == nil {
			continue
		}

		obj := &unstructured.Unstructured{}
		_, gvk, err := decSerializer.Decode(mustMarshalJSON(&rawObj), nil, obj)
		if err != nil {
			return fmt.Errorf("decode to unstructured: %w", err)
		}

		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("find REST mapping for %s: %w", gvk, err)
		}

		var dr dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			ns := obj.GetNamespace()
			if ns == "" {
				ns = "default"
			}
			dr = dynClient.Resource(mapping.Resource).Namespace(ns)
		} else {
			dr = dynClient.Resource(mapping.Resource)
		}

		data := mustMarshalJSON(obj)
		_, err = dr.Patch(ctx, obj.GetName(), "application/apply-patch+yaml", data, metav1.PatchOptions{
			FieldManager: "k3sctl",
		})
		if err != nil {
			return fmt.Errorf("server-side apply %s %s/%s: %w", gvk.Kind, obj.GetNamespace(), obj.GetName(), err)
		}
	}

	return nil
}

func mustMarshalJSON(obj *unstructured.Unstructured) []byte {
	data, err := obj.MarshalJSON()
	if err != nil {
		panic(fmt.Sprintf("marshal unstructured to JSON: %v", err))
	}
	return data
}
