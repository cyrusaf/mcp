package transport_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cyrusaf/mcp/registry"
	"github.com/cyrusaf/mcp/rpc"
	"github.com/cyrusaf/mcp/transport"
)

// helper to start server using HTTP transport
type testHTTPTransport struct {
	server *httptest.Server
	reqCh  chan httpMessage
}

type httpMessage struct {
	req  json.RawMessage
	conn *httpConn
}

type httpConn struct{ ch chan json.RawMessage }

func (c *httpConn) Send(ctx context.Context, resp json.RawMessage) error {
	select {
	case c.ch <- resp:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func newTestHTTPTransport() *testHTTPTransport {
	t := &testHTTPTransport{
		reqCh: make(chan httpMessage, 16),
	}
	t.server = httptest.NewServer(http.HandlerFunc(t.handle))
	return t
}

func (t *testHTTPTransport) handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	conn := &httpConn{ch: make(chan json.RawMessage, 1)}
	msg := httpMessage{req: json.RawMessage(body), conn: conn}
	select {
	case t.reqCh <- msg:
	case <-r.Context().Done():
		return
	}
	resp := <-conn.ch
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
}

func (t *testHTTPTransport) Next(ctx context.Context) (transport.Conn, json.RawMessage, error) {
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case msg, ok := <-t.reqCh:
		if !ok {
			return nil, nil, io.EOF
		}
		return msg.conn, msg.req, nil
	}
}

func (t *testHTTPTransport) Close() error {
	t.server.Close()
	return nil
}

func startHTTPTestServer(tst *testing.T) (addr string, cancel func()) {
	tst.Helper()

	tr := newTestHTTPTransport()
	addr = tr.server.URL

	reg := registry.New()
	registry.RegisterTool(reg, "Echo", func(ctx context.Context, in struct{ Msg string }) (struct{ Msg string }, error) {
		return in, nil
	})

	srv := rpc.NewServer(reg, tr)
	ctx, cancelCtx := context.WithCancel(context.Background())
	go func() { _ = srv.Run(ctx) }()

	cancelFunc := func() {
		cancelCtx()
		_ = tr.Close()
	}
	return addr, cancelFunc
}

func TestHTTPTransportEndToEnd(t *testing.T) {
	url, cancel := startHTTPTestServer(t)
	defer cancel()

	req := struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
	}{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "tools/list"}
	data, _ := json.Marshal(req)

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var out struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  any             `json:"result"`
		Error   *rpc.Error      `json:"error"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Error != nil {
		t.Fatalf("unexpected error: %v", out.Error)
	}
	var result struct {
		Tools []registry.ToolDesc `json:"tools"`
	}
	if b, err := json.Marshal(out.Result); err == nil {
		_ = json.Unmarshal(b, &result)
	}
	if len(result.Tools) != 1 || result.Tools[0].Name != "Echo" {
		t.Fatalf("unexpected tools: %+v", result.Tools)
	}
}
