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
)

type memTransport struct {
	in  chan json.RawMessage
	out chan json.RawMessage
}

func newMemTransport() *memTransport {
	return &memTransport{
		in:  make(chan json.RawMessage, 10),
		out: make(chan json.RawMessage, 10),
	}
}

func (m *memTransport) Next(ctx context.Context) (json.RawMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg, ok := <-m.in:
		if !ok {
			return nil, io.EOF
		}
		return msg, nil
	}
}

func (m *memTransport) Send(ctx context.Context, resp json.RawMessage) error {
	m.out <- resp
	return nil
}

func (m *memTransport) Close() error { return nil }

func startTestServer(t *testing.T) (*Server, *memTransport, context.CancelFunc) {
	tr := newMemTransport()
	reg := registry.New()
	registry.RegisterResource[struct{ ID int }](reg,
		registry.WithURI("res://{id}"),
		registry.WithReadHandler(func(ctx context.Context, uri string) (struct{ ID int }, error) {
			parts := strings.Split(uri, "res://")
			if len(parts) != 2 {
				return struct{ ID int }{}, errors.New("bad uri")
			}
			id, _ := strconv.Atoi(parts[1])
			return struct{ ID int }{ID: id}, nil
		}))
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
	var tools []registry.ToolDesc
	if b, err := json.Marshal(resp.Result); err == nil {
		_ = json.Unmarshal(b, &tools)
	}
	if len(tools) != 1 || tools[0].Name != "Echo" {
		t.Fatalf("unexpected tools: %+v", tools)
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
	var out struct{ Msg string }
	_ = json.Unmarshal(b, &out)
	if out.Msg != "hi" {
		t.Fatalf("unexpected result: %+v", out)
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
	var res []registry.ResourceDesc
	if b, err := json.Marshal(resp.Result); err == nil {
		_ = json.Unmarshal(b, &res)
	}
	if len(res) != 1 || res[0].URI != "res://{id}" {
		t.Fatalf("unexpected resources: %+v", res)
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
	var tools []registry.ToolDesc
	if b, err := json.Marshal(resp.Result); err == nil {
		_ = json.Unmarshal(b, &tools)
	}
	if len(tools) != 1 || tools[0].Description != "echo a message" {
		t.Fatalf("unexpected tools: %+v", tools)
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
	var out struct{ ID int }
	if b, err := json.Marshal(resp.Result); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if out.ID != 42 {
		t.Fatalf("unexpected result: %+v", out)
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
	var out struct {
		Capabilities struct {
			Tools     []registry.ToolDesc     `json:"tools"`
			Resources []registry.ResourceDesc `json:"resources"`
		} `json:"capabilities"`
	}
	if b, err := json.Marshal(resp.Result); err == nil {
		_ = json.Unmarshal(b, &out)
	}
	if len(out.Capabilities.Tools) != 1 || out.Capabilities.Tools[0].Name != "Echo" {
		t.Fatalf("unexpected initialize tools: %+v", out.Capabilities.Tools)
	}
	if len(out.Capabilities.Resources) != 1 || out.Capabilities.Resources[0].URI != "res://{id}" {
		t.Fatalf("unexpected initialize resources: %+v", out.Capabilities.Resources)
	}
}
