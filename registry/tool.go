package registry

import (
	"context"
	"errors"
	"reflect"

	"github.com/cyrusaf/mcp/schema"
)

var ErrInvalidParams = errors.New("invalid params")

type ToolDesc struct {
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	InputSchema  *schema.Schema `json:"input_schema,omitempty"`
	OutputSchema *schema.Schema `json:"output_schema,omitempty"`
	Handler      rawHandler     `json:"-"`
}

type rawHandler interface {
	Req() reflect.Type
	Resp() reflect.Type
	Call(ctx context.Context, req any) (any, error)
}

type ToolOption func(*ToolDesc)

func WithDescription(desc string) ToolOption {
	return func(t *ToolDesc) { t.Description = desc }
}

type handlerFunc[Req any, Resp any] struct {
	f func(context.Context, Req) (Resp, error)
}

func (h *handlerFunc[Req, Resp]) Req() reflect.Type {
	var v Req
	return reflect.TypeOf(v)
}

func (h *handlerFunc[Req, Resp]) Resp() reflect.Type {
	var v Resp
	return reflect.TypeOf(v)
}

func (h *handlerFunc[Req, Resp]) Call(ctx context.Context, req any) (any, error) {
	r, ok := req.(Req)
	if !ok {
		return nil, ErrInvalidParams
	}
	return h.f(ctx, r)
}

func HandlerFunc[Req any, Resp any](fn func(context.Context, Req) (Resp, error)) rawHandler {
	return &handlerFunc[Req, Resp]{f: fn}
}
