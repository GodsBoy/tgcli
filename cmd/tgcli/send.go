package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gotd/td/tg"

	"github.com/GodsBoy/tgcli/internal/format"
)

func newSendCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send messages and files",
	}
	cmd.AddCommand(newSendTextCmd(flags))
	cmd.AddCommand(newSendFileCmd(flags))
	return cmd
}

func newSendTextCmd(flags *rootFlags) *cobra.Command {
	var (
		to      int64
		message string
	)

	cmd := &cobra.Command{
		Use:   "text",
		Short: "Send a text message",
		RunE: func(cmd *cobra.Command, args []string) error {
			if to == 0 || message == "" {
				return fmt.Errorf("--to and --message are required")
			}

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

			if !ac.client.IsAuthed() {
				return fmt.Errorf("not authenticated; run 'tgcli auth' first")
			}

			return ac.client.Run(ctx, func(ctx context.Context, api *tg.Client) error {
				peer := &tg.InputPeerUser{UserID: to}
				_, err := ac.client.SendText(ctx, peer, message)
				if err != nil {
					return fmt.Errorf("send text: %w", err)
				}

				if flags.asJSON {
					return format.WriteJSON(os.Stdout, map[string]interface{}{
						"status": "sent",
						"to":     to,
					})
				}

				fmt.Fprintf(os.Stderr, "Message sent to %d\n", to)
				return nil
			})
		},
	}

	cmd.Flags().Int64Var(&to, "to", 0, "recipient user/chat ID")
	cmd.Flags().StringVar(&message, "message", "", "message text")
	return cmd
}

func newSendFileCmd(flags *rootFlags) *cobra.Command {
	var (
		to      int64
		file    string
		caption string
	)

	cmd := &cobra.Command{
		Use:   "file",
		Short: "Send a file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if to == 0 || file == "" {
				return fmt.Errorf("--to and --file are required")
			}

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

			if !ac.client.IsAuthed() {
				return fmt.Errorf("not authenticated; run 'tgcli auth' first")
			}

			return ac.client.Run(ctx, func(ctx context.Context, api *tg.Client) error {
				// File upload via the Telegram upload API requires the
				// telegram.Uploader, which works at the telegram.Client level
				// (not tg.Client). This is noted as a future enhancement.
				fmt.Fprintf(os.Stderr, "File upload not yet implemented.\n")
				fmt.Fprintf(os.Stderr, "File: %s, To: %d, Caption: %s\n", file, to, caption)

				if flags.asJSON {
					return format.WriteJSON(os.Stdout, map[string]interface{}{
						"status":  "not_implemented",
						"to":      to,
						"file":    file,
						"caption": caption,
					})
				}
				return nil
			})
		},
	}

	cmd.Flags().Int64Var(&to, "to", 0, "recipient user/chat ID")
	cmd.Flags().StringVar(&file, "file", "", "path to file")
	cmd.Flags().StringVar(&caption, "caption", "", "file caption")
	return cmd
}
