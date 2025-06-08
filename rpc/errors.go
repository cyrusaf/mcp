package rpc

import "fmt"

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcError = Error

func ErrorMethodNotFound(method string) *Error {
	return &Error{Code: -32601, Message: fmt.Sprintf("method not found: %s", method)}
}

var ErrInvalidParams = &Error{Code: -32602, Message: "invalid params"}
