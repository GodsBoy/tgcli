package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gotd/td/tg"

	"github.com/GodsBoy/tgcli/internal/format"
	tgsync "github.com/GodsBoy/tgcli/internal/sync"
)

func newSyncCmd(flags *rootFlags) *cobra.Command {
	var follow bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync message history to SQLite (requires prior auth)",
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

			if !ac.client.IsAuthed() {
				return fmt.Errorf("not authenticated; run 'tgcli auth' first")
			}

			return ac.client.Run(ctx, func(ctx context.Context, api *tg.Client) error {
				engine := tgsync.New(ac.client.API(), ac.db)

				result, err := engine.Run(ctx, tgsync.Options{Follow: follow})
				if err != nil {
					return fmt.Errorf("sync: %w", err)
				}

				if flags.asJSON {
					return format.WriteJSON(os.Stdout, map[string]interface{}{
						"messages_stored": result.MessagesStored,
						"chats_stored":    result.ChatsStored,
					})
				}

				fmt.Fprintf(os.Stderr, "Synced %d messages from %d chats\n",
					result.MessagesStored, result.ChatsStored)

				if follow {
					fmt.Fprintln(os.Stderr, "Following for new messages...")
					return engine.Follow(ctx)
				}

				return nil
			})
		},
	}

	cmd.Flags().BoolVar(&follow, "follow", false, "keep syncing new messages (Ctrl+C to stop)")
	return cmd
}
