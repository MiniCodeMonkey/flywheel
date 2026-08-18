// internal/cli/client.go
package cli

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/minicodemonkey/flywheel/internal/config"
	"github.com/minicodemonkey/flywheel/internal/mowl"
)

func newClient(ctx context.Context) (*mowl.Client, config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, cfg, err
	}
	tok, _ := config.LoadToken()
	c := mowl.New(cfg, tok, http.DefaultClient)
	c.SetReauth(func() (string, error) {
		pw := os.Getenv("MOWL_PASSWORD")
		if cfg.Email == "" || pw == "" {
			return "", errors.New("token expired; run `flywheel auth login` (or set MOWL_PASSWORD)")
		}
		nt, err := mowl.Authenticate(ctx, cfg, cfg.Email, pw, http.DefaultClient)
		if err == nil {
			_ = config.SaveToken(nt)
		}
		return nt, err
	})
	return c, cfg, nil
}
