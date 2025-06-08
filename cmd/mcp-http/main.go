package main

import (
	"context"
	"io"
	"log"
	"net/http"

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
	// registry.RegisterResource(api, "User", "users://{id}", UserHandler)
	// registry.RegisterResourceTemplate(api, "User", "users://{id}", UserHandler)
	// registry.RegisterResourceTemplate(api, "Webpage", "webpage://{url}", WebpageHandler,
	//         registry.WithTemplateDescription("load contents of a webpage by URL"))

	registry.RegisterTool(api, "FetchWebpage", WebpageHandler, registry.WithDescription("Fetch contents of a webpage by URL"))

	tr := transport.HTTPTransport(":8080")
	srv := rpc.NewServer(api, tr)
	log.Fatal(srv.Run(context.Background()))
}

func UserHandler(ctx context.Context, id string) (User, error) {
	return User{ID: id}, nil
}

type WebpageReq struct {
	URL string `mcp:"url,primary" json:"url"`
}

type WebpageResp struct {
	Contents string `mcp:"contents" json:"contents"`
}

func WebpageHandler(ctx context.Context, req WebpageReq) (*WebpageResp, error) {
	resp, err := http.Get(req.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &WebpageResp{Contents: string(body)}, nil
}
