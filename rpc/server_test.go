package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/cyrusaf/mcp/registry"
	"github.com/cyrusaf/mcp/transport"
)

type memTransport struct {
	in  chan json.RawMessage
	out chan json.RawMessage
}

type memConn struct{ out chan json.RawMessage }

func (c *memConn) Send(ctx context.Context, resp json.RawMessage) error {
	select {
	case c.out <- resp:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func newMemTransport() *memTransport {
	return &memTransport{
		in:  make(chan json.RawMessage, 10),
		out: make(chan json.RawMessage, 10),
	}
}

func (m *memTransport) Next(ctx context.Context) (transport.Conn, json.RawMessage, error) {
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case msg, ok := <-m.in:
		if !ok {
			return nil, nil, io.EOF
		}
		return &memConn{out: m.out}, msg, nil
	}
}

func (m *memTransport) Close() error { return nil }

func startTestServer(t *testing.T) (*Server, *memTransport, context.CancelFunc) {
	tr := newMemTransport()
	reg := registry.New()
	handler := func(ctx context.Context, uri string) (struct{ ID int }, error) {
		parts := strings.Split(uri, "res://")
		if len(parts) != 2 {
			return struct{ ID int }{}, errors.New("bad uri")
		}
		id, _ := strconv.Atoi(parts[1])
		return struct{ ID int }{ID: id}, nil
	}
	registry.RegisterResource[struct{ ID int }](reg, "Res", "res://{id}", handler)
	registry.RegisterResourceTemplate[struct{ ID int }](reg, "Res", "res://{id}", handler,
		registry.WithTemplateDescription("resource by id"))
	registry.RegisterTool(reg, "Echo", func(ctx context.Context, in struct{ Msg string }) (struct{ Msg string }, error) {
		return in, nil
	}, registry.WithDescription("echo a message"))
	srv := NewServer(reg, tr)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = srv.Run(ctx) }()
	return srv, tr, cancel
}

func TestToolsList(t *testing.T) {
	_, tr, cancel := startTestServer(t)
	defer cancel()

	req := rpcRequest{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "tools/list"}
	data, _ := json.Marshal(req)
	tr.in <- data

	respBytes := <-tr.out
	var resp rpcResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var out struct {
		Tools []registry.ToolDesc `json:"tools"`
	}
	if b, err := json.Marshal(resp.Result); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if len(out.Tools) != 1 || out.Tools[0].Name != "Echo" {
		t.Fatalf("unexpected tools: %+v", out.Tools)
	}
}

func TestToolsCall(t *testing.T) {
	_, tr, cancel := startTestServer(t)
	defer cancel()

	params := callParams{Name: "Echo", Arguments: json.RawMessage(`{"Msg":"hi"}`)}
	pbytes, _ := json.Marshal(params)
	req := rpcRequest{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "tools/call", Params: pbytes}
	data, _ := json.Marshal(req)
	tr.in <- data

	respBytes := <-tr.out
	var resp rpcResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	b, _ := json.Marshal(resp.Result)
	var out struct {
		StructuredContent struct{ Msg string } `json:"structuredContent"`
		Content           []ContentItem        `json:"content"`
	}
	_ = json.Unmarshal(b, &out)
	if out.StructuredContent.Msg != "hi" {
		t.Fatalf("unexpected result: %+v", out.StructuredContent)
	}
	if len(out.Content) != 1 || out.Content[0].Type != "text" || out.Content[0].Data["text"] != "{\"Msg\":\"hi\"}" {
		t.Fatalf("unexpected content: %+v", out.Content)
	}
}

func TestResourcesList(t *testing.T) {
	_, tr, cancel := startTestServer(t)
	defer cancel()

	req := rpcRequest{JSONRPC: "2.0", ID: json.RawMessage(`3`), Method: "resources/list"}
	data, _ := json.Marshal(req)
	tr.in <- data

	respBytes := <-tr.out
	var resp rpcResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var out struct {
		Resources []registry.ResourceDesc `json:"resources"`
	}
	if b, err := json.Marshal(resp.Result); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if len(out.Resources) != 1 || out.Resources[0].URI != "res://{id}" {
		t.Fatalf("unexpected resources: %+v", out.Resources)
	}
}

func TestResourceTemplatesList(t *testing.T) {
	_, tr, cancel := startTestServer(t)
	defer cancel()

	req := rpcRequest{JSONRPC: "2.0", ID: json.RawMessage(`7`), Method: "resources/templates/list"}
	data, _ := json.Marshal(req)
	tr.in <- data

	respBytes := <-tr.out
	var resp rpcResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var out struct {
		ResourceTemplates []registry.ResourceTemplateDesc `json:"resourceTemplates"`
	}
	if b, err := json.Marshal(resp.Result); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if len(out.ResourceTemplates) != 1 || out.ResourceTemplates[0].URITemplate != "res://{id}" {
		t.Fatalf("unexpected resource templates: %+v", out.ResourceTemplates)
	}
}

func TestToolsDescription(t *testing.T) {
	_, tr, cancel := startTestServer(t)
	defer cancel()

	req := rpcRequest{JSONRPC: "2.0", ID: json.RawMessage(`4`), Method: "tools/list"}
	data, _ := json.Marshal(req)
	tr.in <- data

	respBytes := <-tr.out
	var resp rpcResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var out struct {
		Tools []registry.ToolDesc `json:"tools"`
	}
	if b, err := json.Marshal(resp.Result); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if len(out.Tools) != 1 || out.Tools[0].Description != "echo a message" {
		t.Fatalf("unexpected tools: %+v", out.Tools)
	}
}

func TestResourceTemplateDescription(t *testing.T) {
	_, tr, cancel := startTestServer(t)
	defer cancel()

	req := rpcRequest{JSONRPC: "2.0", ID: json.RawMessage(`8`), Method: "resources/templates/list"}
	data, _ := json.Marshal(req)
	tr.in <- data

	respBytes := <-tr.out
	var resp rpcResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var out struct {
		ResourceTemplates []registry.ResourceTemplateDesc `json:"resourceTemplates"`
	}
	if b, err := json.Marshal(resp.Result); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if len(out.ResourceTemplates) != 1 || out.ResourceTemplates[0].Description == nil ||
		*out.ResourceTemplates[0].Description != "resource by id" {
		t.Fatalf("unexpected resource templates: %+v", out.ResourceTemplates)
	}
}

func TestResourceRead(t *testing.T) {
	_, tr, cancel := startTestServer(t)
	defer cancel()

	params := struct {
		URI string `json:"uri"`
	}{URI: "res://42"}
	pbytes, _ := json.Marshal(params)
	req := rpcRequest{JSONRPC: "2.0", ID: json.RawMessage(`5`), Method: "resources/read", Params: pbytes}
	data, _ := json.Marshal(req)
	tr.in <- data

	respBytes := <-tr.out
	var resp rpcResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var out struct {
		Contents []struct {
			URI      string `json:"uri"`
			MimeType string `json:"mime_type"`
			Text     string `json:"text"`
		} `json:"contents"`
	}
	if b, err := json.Marshal(resp.Result); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if len(out.Contents) != 1 {
		t.Fatalf("unexpected result: %+v", out)
	}
	var res struct{ ID int }
	_ = json.Unmarshal([]byte(out.Contents[0].Text), &res)
	if res.ID != 42 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestInitialize(t *testing.T) {
	_, tr, cancel := startTestServer(t)
	defer cancel()

	req := rpcRequest{JSONRPC: "2.0", ID: json.RawMessage(`6`), Method: "initialize"}
	data, _ := json.Marshal(req)
	tr.in <- data

	respBytes := <-tr.out
	var resp rpcResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
	var out InitializeResult
	if b, err := json.Marshal(resp.Result); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if out.ProtocolVersion != "2025-03-26" {
		t.Fatalf("unexpected protocol version: %s", out.ProtocolVersion)
	}
	if out.ServerInfo.Name != "cyrusaf/mcp" || out.ServerInfo.Version != "0.1.0" {
		t.Fatalf("unexpected server info: %+v", out.ServerInfo)
	}
	if out.Capabilities.Tools.ListChanged != false ||
		out.Capabilities.Resources.ListChanged != false ||
		out.Capabilities.Resources.Subscribe != false ||
		out.Capabilities.Prompts.Offered != false {
		t.Fatalf("unexpected capabilities: %+v", out.Capabilities)
	}
}
