package manifest

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

type ResourceDefinition struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Origin string `yaml:"-"`
	Raw    []byte `yaml:"-"`
}

var fnRe = regexp.MustCompile(`[^\w\-\.]`)

func safeFn(input string) string {
	safe := fnRe.ReplaceAllString(input, "_")
	safe = strings.Trim(safe, "._")
	return safe
}

func (rd *ResourceDefinition) Filename() string {
	if strings.HasSuffix(rd.Origin, ".yaml") || strings.HasSuffix(rd.Origin, ".yml") {
		return path.Base(rd.Origin)
	}
	if rd.Metadata.Name != "" {
		return fmt.Sprintf("%s-%s.yaml", safeFn(rd.Kind), safeFn(rd.Metadata.Name))
	}
	return fmt.Sprintf("%s-%s-%d.yaml", safeFn(rd.APIVersion), safeFn(rd.Kind), time.Now().UnixNano())
}

func (rd *ResourceDefinition) Reader() *bytes.Reader {
	return bytes.NewReader(rd.Raw)
}

func (rd *ResourceDefinition) Bytes() []byte {
	return rd.Raw
}

func (rd *ResourceDefinition) Unmarshal(obj any) error {
	if err := yaml.UnmarshalStrict(rd.Bytes(), obj); err != nil {
		return fmt.Errorf("failed to unmarshal %s: %w", rd.Origin, err)
	}
	return nil
}

type Reader struct {
	IgnoreErrors bool
	manifests    []*ResourceDefinition
}

func name(r io.Reader) string {
	type named interface {
		Name() string
	}
	if n, ok := r.(named); ok {
		if nn := n.Name(); nn != "" {
			return nn
		}
	}
	return "manifest"
}

type ParseOption func(*parseOptions)

type parseOptions struct {
	origin string
}

func WithOrigin(origin string) ParseOption {
	return func(po *parseOptions) {
		po.origin = origin
	}
}

func (r *Reader) Parse(input io.Reader, opts ...ParseOption) error {
	po := &parseOptions{}
	for _, opt := range opts {
		opt(po)
	}

	origin := po.origin
	if origin == "" {
		origin = name(input)
	}

	yamlReader := yamlutil.NewYAMLReader(bufio.NewReader(input))

	for {
		rawChunk, err := yamlReader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("error reading input: %w", err)
		}

		if len(bytes.TrimSpace(rawChunk)) == 0 {
			continue
		}

		rd := &ResourceDefinition{}
		if err := yaml.Unmarshal(rawChunk, rd); err != nil {
			if r.IgnoreErrors {
				continue
			}
			return fmt.Errorf("failed to decode resource %s: %w", origin, err)
		}

		if rd.APIVersion == "" || rd.Kind == "" {
			if r.IgnoreErrors {
				continue
			}
			return fmt.Errorf("missing apiVersion or kind in resource %s", origin)
		}

		rd.Origin = origin
		rd.Raw = append([]byte{}, rawChunk...)
		r.manifests = append(r.manifests, rd)
	}

	return nil
}

func (r *Reader) ParseString(input string, opts ...ParseOption) error {
	return r.Parse(strings.NewReader(input), opts...)
}

func (r *Reader) ParseBytes(input []byte, opts ...ParseOption) error {
	return r.Parse(bytes.NewReader(input), opts...)
}

func (r *Reader) ParseBytesWithOrigin(input []byte, origin string) error {
	return r.ParseBytes(input, WithOrigin(origin))
}

func (r *Reader) Resources() []*ResourceDefinition {
	return r.manifests
}

func (r *Reader) Len() int {
	return len(r.manifests)
}

func (r *Reader) FilterResources(filter func(rd *ResourceDefinition) bool) []*ResourceDefinition {
	var resources []*ResourceDefinition
	for _, rd := range r.manifests {
		if filter(rd) {
			resources = append(resources, rd)
		}
	}
	return resources
}

func (r *Reader) GetResources(apiVersion, kind string) ([]*ResourceDefinition, error) {
	resources := r.FilterResources(func(rd *ResourceDefinition) bool {
		return strings.EqualFold(rd.APIVersion, apiVersion) && strings.EqualFold(rd.Kind, kind)
	})

	if len(resources) == 0 {
		return nil, fmt.Errorf("no resources found for apiVersion=%s, kind=%s", apiVersion, kind)
	}
	return resources, nil
}
