package main

import (
	"context"
	"log"

	"github.com/cyrusaf/mcp/registry"
	"github.com/cyrusaf/mcp/rpc"
	"github.com/cyrusaf/mcp/transport"
)

type User struct {
	ID     int    `mcp:"id,primary"`
	Handle string `mcp:"handle,unique"`
}

type CreateUserReq struct{ Handle string }
type CreateUserResp struct{ ID int }

func CreateUser(ctx context.Context, in CreateUserReq) (CreateUserResp, error) {
	return CreateUserResp{ID: 1}, nil
}

func main() {
	api := registry.New()
	registry.RegisterResource[User](api, "User", "users://{id}", func(context.Context, string) (User, error) { return User{}, nil })
	registry.RegisterTool(api, "CreateUser", CreateUser, registry.WithDescription("Create a new user account"))
	srv := rpc.NewServer(api, transport.StdioTransport())
	log.Fatal(srv.Run(context.Background()))
}
