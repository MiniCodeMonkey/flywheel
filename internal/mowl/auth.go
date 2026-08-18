package mowl

import (
	"context"
	"net/http"
	"net/url"

	"github.com/minicodemonkey/flywheel/internal/config"
)

func Authenticate(ctx context.Context, cfg config.Config, email, password string, httpc *http.Client) (string, error) {
	c := New(cfg, "", httpc)
	var ticket string
	err := c.do(ctx, "POST",
		"/v1/Authentication/Authenticate/"+url.PathEscape(cfg.AppPublicKey),
		map[string]string{"Email": email, "Password": password}, &ticket)
	if err != nil {
		return "", err
	}
	var token string
	err = c.do(ctx, "GET",
		"/v1/Authentication/Ticket/"+url.PathEscape(ticket)+"/"+url.PathEscape(cfg.AppPrivateKey),
		nil, &token)
	return token, err
}
