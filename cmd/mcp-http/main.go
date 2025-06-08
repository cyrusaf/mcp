package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/cyrusaf/mcp/registry"
	"github.com/cyrusaf/mcp/rpc"
	"github.com/cyrusaf/mcp/transport"
)

// This example exposes the MCP server over HTTP.
type User struct {
	ID     string `mcp:"id,primary"`
	Handle string `mcp:"handle,unique"`
}

type CreateUserReq struct{ Handle string }
type CreateUserResp struct{ ID int }

func CreateUser(ctx context.Context, in CreateUserReq) (CreateUserResp, error) {
	return CreateUserResp{ID: 1}, nil
}

func main() {
	api := registry.New()
	registry.RegisterResource[User](api, "User", "users://{id}", UserHandler)
	registry.RegisterResourceTemplate[User](api, "User", "users://{id}", UserHandler)
	registry.RegisterResourceTemplate[string](api, "Webpage", "webpage://{url}", WebpageHandler,
		registry.WithTemplateDescription("load contents of a webpage by URL"))

	registry.RegisterTool(api, "CreateUser", CreateUser, registry.WithDescription("Create a new user account"))

	tr := transport.HTTPTransport(":8080")
	srv := rpc.NewServer(api, tr)
	log.Fatal(srv.Run(context.Background()))
}

func UserHandler(ctx context.Context, id string) (User, error) {
	return User{ID: id}, nil
}

func WebpageHandler(ctx context.Context, u string) (string, error) {
	u = strings.TrimPrefix(u, "webpage://")
	decoded, err := url.QueryUnescape(u)
	if err != nil {
		return "", err
	}
	resp, err := http.Get(decoded)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
