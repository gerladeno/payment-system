package pkg

import "context"

type Client struct {
	ID       int
	Name     string
	LimitRPS int
}

type ClientCtxKeyType struct{}

var ClientCtxKey ClientCtxKeyType

func ClientFromCtx(ctx context.Context) *Client {
	if c, ok := ctx.Value(ClientCtxKey).(*Client); ok {
		return c
	}
	return &Client{Name: "unknown", LimitRPS: 1}
}
