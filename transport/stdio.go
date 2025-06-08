package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
)

type Conn interface {
	Send(ctx context.Context, resp json.RawMessage) error
}

type Transport interface {
	Next(ctx context.Context) (Conn, json.RawMessage, error)
	Close() error
}

type stdioTransport struct {
	in  *bufio.Reader
	out io.Writer
}

type stdioConn struct{ out io.Writer }

func (c *stdioConn) Send(ctx context.Context, resp json.RawMessage) error {
	_, err := c.out.Write(append(resp, '\n'))
	return err
}

func StdioTransport() Transport {
	return &stdioTransport{in: bufio.NewReader(os.Stdin), out: os.Stdout}
}

func (s *stdioTransport) Next(ctx context.Context) (Conn, json.RawMessage, error) {
	line, err := s.in.ReadBytes('\n')
	if err != nil {
		return nil, nil, err
	}
	return &stdioConn{out: s.out}, json.RawMessage(line), nil
}

func (s *stdioTransport) Close() error { return nil }
