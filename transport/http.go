package transport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

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

type httpTransport struct {
	srv   *http.Server
	reqCh chan httpMessage
}

// HTTPTransport returns a Transport that serves JSON-RPC requests over HTTP.
// It listens on the provided address.
func HTTPTransport(addr string) Transport {
	tr := &httpTransport{
		reqCh: make(chan httpMessage, 16),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", tr.handle)
	tr.srv = &http.Server{Addr: addr, Handler: mux}
	go tr.srv.ListenAndServe()
	return tr
}

func (h *httpTransport) handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	conn := &httpConn{ch: make(chan json.RawMessage, 1)}
	msg := httpMessage{
		req:  json.RawMessage(body),
		conn: conn,
	}
	select {
	case h.reqCh <- msg:
	case <-r.Context().Done():
		return
	}
	resp := <-conn.ch
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
}

func (h *httpTransport) Next(ctx context.Context) (Conn, json.RawMessage, error) {
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case msg, ok := <-h.reqCh:
		if !ok {
			return nil, nil, io.EOF
		}
		return msg.conn, msg.req, nil
	}
}

func (h *httpTransport) Close() error {
	return h.srv.Shutdown(context.Background())
}
