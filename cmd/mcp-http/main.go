package main

import (
	"context"
	"log"

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

	registry.RegisterTool(api, "CreateUser", CreateUser, registry.WithDescription("Create a new user account"))

	tr := transport.HTTPTransport(":8080")
	srv := rpc.NewServer(api, tr)
	log.Fatal(srv.Run(context.Background()))
}

func UserHandler(ctx context.Context, id string) (User, error) {
	return User{ID: id}, nil
}
