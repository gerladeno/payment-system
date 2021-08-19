package rest

import (
	"context"
	"net/http"
)

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

// TODO proper authentication, add client ID to context
func auth(clientStore ClientStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
