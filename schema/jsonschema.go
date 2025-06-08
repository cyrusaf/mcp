package schema

import "reflect"

type Schema struct {
	Type       string             `json:"type,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
}

func ReflectFromType(t reflect.Type) *Schema {
	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}
	case reflect.Slice, reflect.Array:
		return &Schema{Type: "array", Items: ReflectFromType(t.Elem())}
	case reflect.Struct:
		props := make(map[string]*Schema)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" { // unexported
				continue
			}
			props[f.Name] = ReflectFromType(f.Type)
		}
		return &Schema{Type: "object", Properties: props}
	default:
		return &Schema{Type: "object"}
	}
}

func Reflect(v any) *Schema { return ReflectFromType(reflect.TypeOf(v)) }
