// internal/cli/auth.go
package cli

import (
	"fmt"
	"net/http"
	"os"

	"github.com/minicodemonkey/flywheel/internal/config"
	"github.com/minicodemonkey/flywheel/internal/mowl"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage MOWL authentication",
	}
	cmd.AddCommand(newAuthLoginCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var email string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with MOWL and save a token",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if email == "" {
				email = cfg.Email
			}
			if email == "" {
				fmt.Fprint(cmd.OutOrStdout(), "Email: ")
				fmt.Fscanln(cmd.InOrStdin(), &email)
			}
			pw := os.Getenv("MOWL_PASSWORD")
			if pw == "" {
				fmt.Fprint(cmd.OutOrStdout(), "Password: ")
				b, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Fprintln(cmd.OutOrStdout())
				if err != nil {
					return fmt.Errorf("read password: %w", err)
				}
				pw = string(b)
			}
			tok, err := mowl.Authenticate(cmd.Context(), cfg, email, pw, http.DefaultClient)
			if err != nil {
				return fmt.Errorf("authenticate: %w", err)
			}
			if err := config.SaveToken(tok); err != nil {
				return fmt.Errorf("save token: %w", err)
			}
			cfg.Email = email
			if err := cfg.Save(); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged in as", email)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "MOWL account email (defaults to saved config or prompts)")
	return cmd
}
