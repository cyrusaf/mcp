package registry

import (
	"context"
	"reflect"
	"strings"
	"sync"

	"github.com/cyrusaf/mcp/schema"
)

type Registry struct {
	mu        sync.RWMutex
	resources []*ResourceDesc
	tools     []*ToolDesc
}

func New() *Registry { return &Registry{} }

func RegisterResource[T any](r *Registry, opts ...ResourceOption) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	desc := &ResourceDesc{}
	for _, opt := range opts {
		opt(desc)
	}
	var zero T
	if desc.JSONSchema == nil {
		if desc.Handler != nil {
			desc.JSONSchema = schema.ReflectFromType(desc.Handler.Resp())
		} else {
			desc.JSONSchema = schema.ReflectFromType(reflect.TypeOf(zero))
		}
	}
	r.resources = append(r.resources, desc)
	return r
}

func RegisterTool[Req any, Resp any](r *Registry, name string, fn func(context.Context, Req) (Resp, error), opts ...ToolOption) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	desc := &ToolDesc{Name: name, Handler: HandlerFunc(fn)}
	for _, opt := range opts {
		opt(desc)
	}
	desc.InputSchema = schema.ReflectFromType(desc.Handler.Req())
	desc.OutputSchema = schema.ReflectFromType(desc.Handler.Resp())
	r.tools = append(r.tools, desc)
	return r
}

func (r *Registry) Tools() []*ToolDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ToolDesc, len(r.tools))
	for i, t := range r.tools {
		clone := *t
		clone.Handler = nil
		out[i] = &clone
	}
	return out
}

func (r *Registry) ToolsMap() map[string]*ToolDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]*ToolDesc, len(r.tools))
	for _, t := range r.tools {
		clone := *t
		clone.Handler = nil
		out[t.Name] = &clone
	}
	return out
}

func (r *Registry) Resources() []*ResourceDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ResourceDesc, len(r.resources))
	for i, res := range r.resources {
		clone := *res
		out[i] = &clone
	}
	return out
}

func (r *Registry) ResourcesMap() map[string]*ResourceDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]*ResourceDesc, len(r.resources))
	for _, res := range r.resources {
		clone := *res
		out[res.URI] = &clone
	}
	return out
}

func (r *Registry) findResource(uri string) *ResourceDesc {
	for _, res := range r.resources {
		tmpl := res.URI
		if tmpl == uri {
			return res
		}
		if i := strings.Index(tmpl, "{"); i > 0 {
			prefix := tmpl[:i]
			if strings.HasPrefix(uri, prefix) {
				return res
			}
		}
	}
	return nil
}

func (r *Registry) FindResource(uri string) *ResourceDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.findResource(uri)
}

func (r *Registry) findTool(name string) *ToolDesc {
	for _, t := range r.tools {
		if t.Name == name {
			return t
		}
	}
	return nil
}

func (r *Registry) FindTool(name string) *ToolDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.findTool(name)
}
