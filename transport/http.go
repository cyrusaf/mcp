package transport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

type httpMessage struct {
	id     string
	req    json.RawMessage
	respCh chan json.RawMessage
}

type httpTransport struct {
	srv     *http.Server
	reqCh   chan httpMessage
	mu      sync.Mutex
	pending map[string]chan json.RawMessage
}

// HTTPTransport returns a Transport that serves JSON-RPC requests over HTTP.
// It listens on the provided address.
func HTTPTransport(addr string) Transport {
	tr := &httpTransport{
		reqCh:   make(chan httpMessage, 16),
		pending: make(map[string]chan json.RawMessage),
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
	var tmp struct {
		ID json.RawMessage `json:"id"`
	}
	_ = json.Unmarshal(body, &tmp)
	msg := httpMessage{
		id:     string(tmp.ID),
		req:    json.RawMessage(body),
		respCh: make(chan json.RawMessage, 1),
	}
	select {
	case h.reqCh <- msg:
	case <-r.Context().Done():
		return
	}
	resp := <-msg.respCh
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
}

func (h *httpTransport) Next(ctx context.Context) (json.RawMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg, ok := <-h.reqCh:
		if !ok {
			return nil, io.EOF
		}
		if msg.id != "" {
			h.mu.Lock()
			h.pending[msg.id] = msg.respCh
			h.mu.Unlock()
		}
		return msg.req, nil
	}
}

func (h *httpTransport) Send(ctx context.Context, resp json.RawMessage) error {
	var tmp struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(resp, &tmp); err != nil {
		return err
	}
	id := string(tmp.ID)
	h.mu.Lock()
	ch, ok := h.pending[id]
	if ok {
		delete(h.pending, id)
	}
	h.mu.Unlock()
	if !ok {
		return nil
	}
	select {
	case ch <- resp:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *httpTransport) Close() error {
	return h.srv.Shutdown(context.Background())
}
