package rpc

import (
	"encoding/json"
	"fmt"
)

// ContentItem represents a single block in the `content` array.
// The keys of Data are flattened alongside the Type when marshalled.
type ContentItem struct {
	Type string
	Data map[string]any
}

func (c ContentItem) MarshalJSON() ([]byte, error) {
	m := make(map[string]any, len(c.Data)+1)
	for k, v := range c.Data {
		m[k] = v
	}
	m["type"] = c.Type
	return json.Marshal(m)
}

func (c *ContentItem) UnmarshalJSON(b []byte) error {
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	t, ok := m["type"].(string)
	if !ok {
		return fmt.Errorf("content item missing type")
	}
	c.Type = t
	delete(m, "type")
	c.Data = m
	return nil
}

// NewTextContent returns a ContentItem with type "text".
func NewTextContent(text string) ContentItem {
	return ContentItem{Type: "text", Data: map[string]any{"text": text}}
}
