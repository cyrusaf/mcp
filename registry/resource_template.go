package registry

import (
	"context"
	"reflect"

	"github.com/cyrusaf/mcp/schema"
)

type ResourceTemplateDesc struct {
	Name        string             `json:"name"`
	URITemplate string             `json:"uriTemplate"`
	JSONSchema  *schema.Schema     `json:"json_schema,omitempty"`
	Handler     rawResourceHandler `json:"-"`
}

type rawResourceTemplateHandler interface {
	Resp() reflect.Type
	Read(ctx context.Context, uri string) (any, error)
}

type resourceTemplateHandlerFunc[Resp any] struct {
	f func(context.Context, string) (Resp, error)
}

func (h *resourceTemplateHandlerFunc[Resp]) Resp() reflect.Type {
	var v Resp
	return reflect.TypeOf(v)
}

func (h *resourceTemplateHandlerFunc[Resp]) Read(ctx context.Context, uri string) (any, error) {
	return h.f(ctx, uri)
}

func ResourceTemplateHandlerFunc[Resp any](fn func(context.Context, string) (Resp, error)) rawResourceHandler {
	return &resourceHandlerFunc[Resp]{f: fn}
}

type ResourceTemplateOption func(*ResourceTemplateDesc)

func WithTemplateSchema(s *schema.Schema) ResourceTemplateOption {
	return func(r *ResourceTemplateDesc) { r.JSONSchema = s }
}
