package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/cyrusaf/mcp/registry"
	"github.com/cyrusaf/mcp/transport"
)

// Server dispatches JSON-RPC requests

type Server struct {
	reg *registry.Registry
	tr  transport.Transport
}

func NewServer(reg *registry.Registry, tr transport.Transport) *Server {
	return &Server{reg: reg, tr: tr}
}

func (s *Server) Run(ctx context.Context) error {
	for {
		conn, raw, err := s.tr.Next(ctx)
		if err != nil {
			return err
		}
		go s.handle(ctx, conn, raw)
	}
}

func (s *Server) handle(ctx context.Context, conn transport.Conn, raw json.RawMessage) {
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		s.sendError(ctx, conn, nil, ErrInvalidParams)
		return
	}

	switch req.Method {
	case "initialize":
		var res InitializeResult
		res.ProtocolVersion = "2025-03-26"
		res.ServerInfo.Name = "cyrusaf/mcp"
		res.ServerInfo.Version = "0.1.0"
		// all capability flags default to false
		s.send(ctx, conn, req.ID, res)
	case "tools/list":
		s.send(ctx, conn, req.ID, map[string]any{
			"tools": s.reg.Tools(),
		})
	case "resources/list":
		s.send(ctx, conn, req.ID, map[string]any{
			"resources": s.reg.Resources(),
		})
	case "resources/templates/list":
		s.send(ctx, conn, req.ID, map[string]any{
			"resourceTemplates": s.reg.ResourceTemplates(),
		})
	case "tools/call":
		s.handleToolCall(ctx, conn, req)
	case "resources/read":
		s.handleResourceRead(ctx, conn, req)
	default:
		s.sendError(ctx, conn, req.ID, ErrorMethodNotFound(req.Method))
	}
}

func (s *Server) send(ctx context.Context, conn transport.Conn, id json.RawMessage, result any) {
	resp := rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
	data, _ := json.Marshal(resp)
	data = append(data, '\n')
	_ = conn.Send(ctx, data)
}

func (s *Server) sendError(ctx context.Context, conn transport.Conn, id json.RawMessage, err *Error) {
	resp := rpcResponse{JSONRPC: "2.0", ID: id, Error: err}
	data, _ := json.Marshal(resp)
	_ = conn.Send(ctx, data)
}

// tools/call params structure

type callParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type toolStructuredResp struct {
	StructuredContent any           `json:"structuredContent,omitempty"`
	Content           []ContentItem `json:"content,omitempty"`
}

func (s *Server) handleToolCall(ctx context.Context, conn transport.Conn, req rpcRequest) {
	var params callParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(ctx, conn, req.ID, ErrInvalidParams)
		return
	}
	tool := s.reg.FindTool(params.Name)
	if tool == nil {
		s.sendError(ctx, conn, req.ID, ErrorMethodNotFound(params.Name))
		return
	}
	// decode arguments
	arg := reflect.New(tool.Handler.Req()).Interface()
	if len(params.Arguments) > 0 {
		if err := json.Unmarshal(params.Arguments, &arg); err != nil {
			s.sendError(ctx, conn, req.ID, ErrInvalidParams)
			return
		}
	}
	val, err := tool.Handler.Call(ctx, reflect.ValueOf(arg).Elem().Interface())
	if err != nil {
		s.sendError(ctx, conn, req.ID, &Error{Code: -32000, Message: err.Error()})
		return
	}

	var resp toolStructuredResp
	if tool.OutputSchema != nil {
		resp.StructuredContent = val
		if b, err := json.Marshal(val); err == nil {
			resp.Content = []ContentItem{
				NewTextContent(string(b)),
			}
		}
	} else {
		resp.Content = []ContentItem{
			{Type: "text", Data: map[string]any{"text": fmt.Sprint(val)}},
		}
	}
	s.send(ctx, conn, req.ID, resp)
}

func (s *Server) handleResourceRead(ctx context.Context, conn transport.Conn, req rpcRequest) {
	type params struct {
		URI  string          `json:"uri"`
		Meta json.RawMessage `json:"_meta,omitempty"`
	}
	var p params
	if err := json.Unmarshal(req.Params, &p); err != nil || p.URI == "" {
		s.sendError(ctx, conn, req.ID, ErrInvalidParams)
		return
	}
	handler := s.reg.FindResource(p.URI)
	if handler == nil {
		s.sendError(ctx, conn, req.ID, ErrorMethodNotFound(p.URI))
		return
	}
	val, err := handler.Read(ctx, p.URI)
	if err != nil {
		s.sendError(ctx, conn, req.ID, &Error{Code: -32000, Message: err.Error()})
		return
	}
	valJSON, err := json.Marshal(val)
	if err != nil {
		s.sendError(ctx, conn, req.ID, &Error{Code: -32000, Message: err.Error()})
		return
	}

	out := map[string]any{
		"contents": []any{
			map[string]any{
				"uri":       p.URI + "#json",
				"mime_type": "application/json",
				"text":      string(valJSON),
			},
		},
	}
	s.send(ctx, conn, req.ID, out)
}
