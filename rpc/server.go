package rpc

import (
	"context"
	"encoding/json"
	"net/url"
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
		type capabilities struct {
			Tools     []*registry.ToolDesc     `json:"tools"`
			Resources []*registry.ResourceDesc `json:"resources"`
		}
		type initializeResult struct {
			Capabilities capabilities `json:"capabilities"`
		}
		res := initializeResult{
			Capabilities: capabilities{
				Tools:     s.reg.Tools(),
				Resources: s.reg.Resources(),
			},
		}
		s.send(ctx, conn, req.ID, res)
	case "tools/list":
		s.send(ctx, conn, req.ID, s.reg.Tools())
	case "resources/list":
		s.send(ctx, conn, req.ID, s.reg.Resources())
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
	s.send(ctx, conn, req.ID, val)
}

func (s *Server) handleResourceRead(ctx context.Context, conn transport.Conn, req rpcRequest) {
	type params struct {
		URI string `json:"uri"`
	}
	var p params
	if err := json.Unmarshal(req.Params, &p); err != nil || p.URI == "" {
		s.sendError(ctx, conn, req.ID, ErrInvalidParams)
		return
	}
	if _, err := url.Parse(p.URI); err != nil {
		s.sendError(ctx, conn, req.ID, ErrInvalidParams)
		return
	}
	res := s.reg.FindResource(p.URI)
	if res == nil || res.Handler == nil {
		s.sendError(ctx, conn, req.ID, ErrorMethodNotFound(p.URI))
		return
	}
	val, err := res.Handler.Read(ctx, p.URI)
	if err != nil {
		s.sendError(ctx, conn, req.ID, &Error{Code: -32000, Message: err.Error()})
		return
	}
	s.send(ctx, conn, req.ID, val)
}
