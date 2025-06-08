package registry

import (
	"context"
	"reflect"
	"strings"
	"sync"

	"github.com/cyrusaf/mcp/schema"
)

type Registry struct {
	mu                sync.RWMutex
	resources         map[string]*ResourceDesc
	resourceTemplates map[string]*ResourceTemplateDesc
	tools             map[string]*ToolDesc
}

func New() *Registry {
	return &Registry{
		resources:         make(map[string]*ResourceDesc),
		resourceTemplates: make(map[string]*ResourceTemplateDesc),
		tools:             make(map[string]*ToolDesc),
	}
}

func RegisterResource[T any](r *Registry, name, uri string, handler func(context.Context, string) (T, error), opts ...ResourceOption) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	desc := &ResourceDesc{Name: name, URI: uri, Handler: ResourceHandlerFunc(handler)}
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
	if r.resources == nil {
		r.resources = make(map[string]*ResourceDesc)
	}
	r.resources[uri] = desc
	return r
}

func RegisterResourceTemplate[T any](r *Registry, name, uriTemplate string, handler func(context.Context, string) (T, error), opts ...ResourceTemplateOption) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	desc := &ResourceTemplateDesc{Name: name, URITemplate: uriTemplate, Handler: ResourceHandlerFunc(handler)}
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
	if r.resourceTemplates == nil {
		r.resourceTemplates = make(map[string]*ResourceTemplateDesc)
	}
	r.resourceTemplates[uriTemplate] = desc
	return r
}

func RegisterTool[Req any, Resp any](r *Registry, name string, fn func(context.Context, Req) (Resp, error), opts ...ToolOption) *Registry {
	r.mu.Lock()
	defer r.mu.Unlock()
	desc := &ToolDesc{Name: name, Handler: HandlerFunc(fn)}
	for _, opt := range opts {
		opt(desc)
	}
	desc.InputSchema = *schema.ReflectFromType(desc.Handler.Req())
	desc.OutputSchema = schema.ReflectFromType(desc.Handler.Resp())
	if r.tools == nil {
		r.tools = make(map[string]*ToolDesc)
	}
	r.tools[name] = desc
	return r
}

func (r *Registry) Tools() []*ToolDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ToolDesc, 0, len(r.tools))
	for _, t := range r.tools {
		clone := *t
		clone.Handler = nil
		out = append(out, &clone)
	}
	return out
}

func (r *Registry) ToolsMap() map[string]*ToolDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]*ToolDesc, len(r.tools))
	for name, t := range r.tools {
		clone := *t
		clone.Handler = nil
		out[name] = &clone
	}
	return out
}

func (r *Registry) Resources() []*ResourceDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ResourceDesc, 0, len(r.resources))
	for _, res := range r.resources {
		clone := *res
		out = append(out, &clone)
	}
	return out
}

func (r *Registry) ResourceTemplates() []*ResourceTemplateDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*ResourceTemplateDesc, 0, len(r.resourceTemplates))
	for _, res := range r.resourceTemplates {
		clone := *res
		clone.Handler = nil
		out = append(out, &clone)
	}
	return out
}

func (r *Registry) ResourcesMap() map[string]*ResourceDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]*ResourceDesc, len(r.resources))
	for uri, res := range r.resources {
		clone := *res
		out[uri] = &clone
	}
	return out
}

func (r *Registry) findResource(uri string) rawResourceHandler {
	if res, ok := r.resources[uri]; ok {
		return res.Handler
	}
	for _, res := range r.resourceTemplates {
		tmpl := res.URITemplate
		if i := strings.Index(tmpl, "{"); i > 0 {
			prefix := tmpl[:i]
			if strings.HasPrefix(uri, prefix) {
				return res.Handler
			}
		} else if tmpl == uri {
			return res.Handler
		}
	}
	return nil
}

func (r *Registry) FindResource(uri string) rawResourceHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.findResource(uri)
}

func (r *Registry) findTool(name string) *ToolDesc {
	if t, ok := r.tools[name]; ok {
		return t
	}
	return nil
}

func (r *Registry) FindTool(name string) *ToolDesc {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.findTool(name)
}
