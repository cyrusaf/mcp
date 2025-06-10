package rpc

import "encoding/json"

// ResourceReadParams represents parameters to the "resources/read" JSON-RPC call.
type ResourceReadParams struct {
	URI  string          `json:"uri"`
	Meta json.RawMessage `json:"_meta,omitempty"`
}

// ResourceContent represents an item in a resource read response.
type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// ResourceReadResult represents the result payload of the "resources/read" call.
type ResourceReadResult struct {
	Contents []ResourceContent `json:"contents"`
}
