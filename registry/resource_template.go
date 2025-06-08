package registry

import (
	"github.com/cyrusaf/mcp/schema"
)

type ResourceTemplateDesc struct {
	Name        string             `json:"name"`
	URITemplate string             `json:"uriTemplate"`
	JSONSchema  *schema.Schema     `json:"json_schema,omitempty"`
	Description *string            `json:"description,omitempty"`
	Handler     rawResourceHandler `json:"-"`
}

type ResourceTemplateOption func(*ResourceTemplateDesc)

func WithTemplateSchema(s *schema.Schema) ResourceTemplateOption {
	return func(r *ResourceTemplateDesc) { r.JSONSchema = s }
}

func WithTemplateDescription(desc string) ResourceTemplateOption {
	return func(r *ResourceTemplateDesc) { r.Description = &desc }
}
