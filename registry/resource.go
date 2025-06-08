package registry

import (
	"context"
	"reflect"

	"github.com/cyrusaf/mcp/schema"
)

type ResourceDesc struct {
	Name       string             `json:"name"`
	URI        string             `json:"uri"`
	JSONSchema *schema.Schema     `json:"json_schema,omitempty"`
	Handler    rawResourceHandler `json:"-"`
}

type rawResourceHandler interface {
	Resp() reflect.Type
	Read(ctx context.Context, uri string) (any, error)
}

type resourceHandlerFunc[Resp any] struct {
	f func(context.Context, string) (Resp, error)
}

func (h *resourceHandlerFunc[Resp]) Resp() reflect.Type {
	var v Resp
	return reflect.TypeOf(v)
}

func (h *resourceHandlerFunc[Resp]) Read(ctx context.Context, uri string) (any, error) {
	return h.f(ctx, uri)
}

func ResourceHandlerFunc[Resp any](fn func(context.Context, string) (Resp, error)) rawResourceHandler {
	return &resourceHandlerFunc[Resp]{f: fn}
}

type ResourceOption func(*ResourceDesc)

func WithSchema(s *schema.Schema) ResourceOption {
	return func(r *ResourceDesc) { r.JSONSchema = s }
}
