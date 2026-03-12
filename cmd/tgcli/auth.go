package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"

	tgauth "github.com/GodsBoy/tgcli/internal/auth"
	"github.com/GodsBoy/tgcli/internal/format"
)

func newAuthCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Telegram (phone + OTP + optional 2FA)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), flags.timeout)
			defer cancel()

			ac, err := newAppContext(ctx, flags, true)
			if err != nil {
				return err
			}
			defer ac.close()

			if err := ac.initClient(); err != nil {
				return err
			}

			return ac.client.Run(ctx, func(ctx context.Context, api *tg.Client) error {
				authClient := auth.NewClient(api, rand.Reader, ac.cfg.AppID, ac.cfg.AppHash)

				flow := &tgauth.Flow{
					Phone:  ac.cfg.Phone,
					Stdin:  os.Stdin,
					Stderr: os.Stderr,
				}

				if err := flow.Run(ctx, authClient); err != nil {
					return fmt.Errorf("auth flow: %w", err)
				}

				user, err := tgauth.CheckAuthorization(ctx, api)
				if err != nil {
					return fmt.Errorf("check auth: %w", err)
				}

				result := map[string]interface{}{
					"status":     "authenticated",
					"user_id":    user.ID,
					"first_name": user.FirstName,
					"last_name":  user.LastName,
					"username":   user.Username,
					"phone":      user.Phone,
				}

				if flags.asJSON {
					return format.WriteJSON(os.Stdout, result)
				}

				fmt.Fprintf(os.Stderr, "Authenticated as %s %s (@%s)\n",
					user.FirstName, user.LastName, user.Username)
				return nil
			})
		},
	}

	cmd.AddCommand(newAuthStatusCmd(flags))
	cmd.AddCommand(newAuthLogoutCmd(flags))
	return cmd
}

func newAuthStatusCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), flags.timeout)
			defer cancel()

			ac, err := newAppContext(ctx, flags, false)
			if err != nil {
				return err
			}
			defer ac.close()

			if err := ac.initClient(); err != nil {
				return err
			}

			if !ac.client.IsAuthed() {
				result := map[string]interface{}{
					"status": "not_authenticated",
				}
				if flags.asJSON {
					return format.WriteJSON(os.Stdout, result)
				}
				fmt.Println("Not authenticated. Run 'tgcli auth' to log in.")
				return nil
			}

			var user map[string]interface{}
			runErr := ac.client.Run(ctx, func(ctx context.Context, api *tg.Client) error {
				u, err := tgauth.CheckAuthorization(ctx, api)
				if err != nil {
					return err
				}
				user = map[string]interface{}{
					"status":     "authenticated",
					"user_id":    u.ID,
					"first_name": u.FirstName,
					"last_name":  u.LastName,
					"username":   u.Username,
					"phone":      u.Phone,
				}
				return nil
			})
			if runErr != nil {
				return runErr
			}

			if flags.asJSON {
				return format.WriteJSON(os.Stdout, user)
			}

			fmt.Printf("Authenticated as %s %s (@%s)\n",
				user["first_name"], user["last_name"], user["username"])
			return nil
		},
	}
}

func newAuthLogoutCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Invalidate the current session",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), flags.timeout)
			defer cancel()

			ac, err := newAppContext(ctx, flags, true)
			if err != nil {
				return err
			}
			defer ac.close()

			if err := ac.initClient(); err != nil {
				return err
			}

			return ac.client.Run(ctx, func(ctx context.Context, api *tg.Client) error {
				if err := tgauth.Logout(ctx, api); err != nil {
					return fmt.Errorf("logout: %w", err)
				}

				os.Remove(ac.client.SessionPath())

				if flags.asJSON {
					return format.WriteJSON(os.Stdout, map[string]string{"status": "logged_out"})
				}
				fmt.Fprintln(os.Stderr, "Logged out successfully.")
				return nil
			})
		},
	}
}
