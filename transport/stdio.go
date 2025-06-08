package transport

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
)

type Transport interface {
	Next(ctx context.Context) (json.RawMessage, error)
	Send(ctx context.Context, resp json.RawMessage) error
	Close() error
}

type stdioTransport struct {
	in  *bufio.Reader
	out io.Writer
}

func StdioTransport() Transport {
	return &stdioTransport{in: bufio.NewReader(os.Stdin), out: os.Stdout}
}

func (s *stdioTransport) Next(ctx context.Context) (json.RawMessage, error) {
	line, err := s.in.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return json.RawMessage(line), nil
}

func (s *stdioTransport) Send(ctx context.Context, resp json.RawMessage) error {
	_, err := s.out.Write(append(resp, '\n'))
	return err
}

func (s *stdioTransport) Close() error { return nil }
