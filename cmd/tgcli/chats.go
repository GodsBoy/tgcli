package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/GodsBoy/tgcli/internal/format"
)

func newChatsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chats",
		Short: "Manage chats",
	}
	cmd.AddCommand(newChatsListCmd(flags))
	return cmd
}

func newChatsListCmd(flags *rootFlags) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all chats",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := newAppContext(cmd.Context(), flags, false)
			if err != nil {
				return err
			}
			defer ac.close()

			chats, err := ac.db.ListChats(limit)
			if err != nil {
				return fmt.Errorf("list chats: %w", err)
			}

			if flags.asJSON {
				return format.WriteJSON(os.Stdout, chats)
			}

			if len(chats) == 0 {
				fmt.Println("No chats found. Run 'tgcli sync' first.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "CHAT_ID\tKIND\tNAME\tLAST_MESSAGE")
			for _, c := range chats {
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\n",
					c.ChatID, c.Kind, c.Name,
					c.LastMessageTS.Format("2006-01-02 15:04"))
			}
			w.Flush()
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100, "max number of chats")
	return cmd
}
