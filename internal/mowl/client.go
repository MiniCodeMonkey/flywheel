package mowl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/minicodemonkey/flywheel/internal/config"
)

type Client struct {
	cfg    config.Config
	token  string
	http   *http.Client
	reauth func() (string, error)
}

func New(cfg config.Config, token string, httpc *http.Client) *Client {
	if httpc == nil {
		httpc = http.DefaultClient
	}
	return &Client{cfg: cfg, token: token, http: httpc}
}

func (c *Client) SetReauth(fn func() (string, error)) { c.reauth = fn }

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	resp, status, err := c.roundtrip(ctx, method, path, body)
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized && c.reauth != nil {
		tok, rerr := c.reauth()
		if rerr != nil {
			return fmt.Errorf("mowl: reauth: %w", rerr)
		}
		c.token = tok
		resp, status, err = c.roundtrip(ctx, method, path, body)
		if err != nil {
			return fmt.Errorf("mowl: %w", err)
		}
	}
	var env envelope
	if len(resp) > 0 {
		if err := json.Unmarshal(resp, &env); err != nil {
			return fmt.Errorf("mowl: decode (status %d): %w", status, err)
		}
	}
	if env.Error != nil {
		return fmt.Errorf("mowl: %s", env.Error.Message)
	}
	if status >= 400 {
		return fmt.Errorf("mowl: http %d", status)
	}
	if out != nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}

func (c *Client) roundtrip(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.cfg.APIBase+path, rdr)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", c.token)
	req.Header.Set("itc-client-os-family", "MacOS")
	req.Header.Set("itc-client-version", c.cfg.ClientVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	return data, res.StatusCode, err
}
