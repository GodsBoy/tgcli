package main

import (
	"database/sql"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/GodsBoy/tgcli/internal/format"
	"github.com/GodsBoy/tgcli/internal/store"
)

func newMessagesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "messages",
		Short: "List, search, and show messages",
	}
	cmd.AddCommand(newMessagesListCmd(flags))
	cmd.AddCommand(newMessagesSearchCmd(flags))
	cmd.AddCommand(newMessagesShowCmd(flags))
	return cmd
}

func newMessagesListCmd(flags *rootFlags) *cobra.Command {
	var (
		chatID int64
		limit  int
		after  string
		before string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messages in a chat",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := newAppContext(cmd.Context(), flags, false)
			if err != nil {
				return err
			}
			defer ac.close()

			params := store.ListMessagesParams{
				ChatID: chatID,
				Limit:  limit,
			}
			if after != "" {
				t, err := time.Parse(time.RFC3339, after)
				if err != nil {
					return fmt.Errorf("invalid --after time: %w", err)
				}
				params.After = t
			}
			if before != "" {
				t, err := time.Parse(time.RFC3339, before)
				if err != nil {
					return fmt.Errorf("invalid --before time: %w", err)
				}
				params.Before = t
			}

			msgs, err := ac.db.ListMessages(params)
			if err != nil {
				return fmt.Errorf("list messages: %w", err)
			}

			if flags.asJSON {
				return format.WriteJSON(os.Stdout, msgs)
			}

			printMessages(msgs)
			return nil
		},
	}

	cmd.Flags().Int64Var(&chatID, "chat", 0, "chat ID to list messages for")
	cmd.Flags().IntVar(&limit, "limit", 50, "max number of messages")
	cmd.Flags().StringVar(&after, "after", "", "show messages after this time (RFC3339)")
	cmd.Flags().StringVar(&before, "before", "", "show messages before this time (RFC3339)")
	return cmd
}

func newMessagesSearchCmd(flags *rootFlags) *cobra.Command {
	var (
		chatID int64
		limit  int
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Full-text search messages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := newAppContext(cmd.Context(), flags, false)
			if err != nil {
				return err
			}
			defer ac.close()

			msgs, err := ac.db.SearchMessages(store.SearchMessagesParams{
				Query:  args[0],
				ChatID: chatID,
				Limit:  limit,
			})
			if err != nil {
				return fmt.Errorf("search: %w", err)
			}

			if flags.asJSON {
				return format.WriteJSON(os.Stdout, msgs)
			}

			if len(msgs) == 0 {
				fmt.Println("No results found.")
				return nil
			}

			printMessages(msgs)
			return nil
		},
	}

	cmd.Flags().Int64Var(&chatID, "chat", 0, "restrict search to a chat")
	cmd.Flags().IntVar(&limit, "limit", 50, "max results")
	return cmd
}

func newMessagesShowCmd(flags *rootFlags) *cobra.Command {
	var (
		chatID int64
		msgID  int
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show a single message",
		RunE: func(cmd *cobra.Command, args []string) error {
			if chatID == 0 || msgID == 0 {
				return fmt.Errorf("--chat and --id are required")
			}

			ac, err := newAppContext(cmd.Context(), flags, false)
			if err != nil {
				return err
			}
			defer ac.close()

			msg, err := ac.db.GetMessage(chatID, msgID)
			if err == sql.ErrNoRows {
				return fmt.Errorf("message not found (chat=%d, id=%d)", chatID, msgID)
			}
			if err != nil {
				return fmt.Errorf("get message: %w", err)
			}

			if flags.asJSON {
				return format.WriteJSON(os.Stdout, msg)
			}

			fmt.Printf("Chat:      %d (%s)\n", msg.ChatID, msg.ChatName)
			fmt.Printf("Message:   %d\n", msg.MsgID)
			fmt.Printf("Sender:    %d\n", msg.SenderID)
			fmt.Printf("Time:      %s\n", msg.Timestamp.Format(time.RFC3339))
			fmt.Printf("From me:   %v\n", msg.FromMe)
			if msg.Text != "" {
				fmt.Printf("Text:      %s\n", msg.Text)
			}
			if msg.MediaType != "" {
				fmt.Printf("Media:     %s\n", msg.MediaType)
			}
			if msg.MediaCaption != "" {
				fmt.Printf("Caption:   %s\n", msg.MediaCaption)
			}
			if msg.ReplyToMsgID != 0 {
				fmt.Printf("Reply to:  %d\n", msg.ReplyToMsgID)
			}
			return nil
		},
	}

	cmd.Flags().Int64Var(&chatID, "chat", 0, "chat ID")
	cmd.Flags().IntVar(&msgID, "id", 0, "message ID")
	return cmd
}

func printMessages(msgs []store.Message) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "CHAT\tMSG_ID\tSENDER\tTIME\tTEXT")
	for _, m := range msgs {
		text := m.Text
		if text == "" && m.MediaCaption != "" {
			text = "[" + m.MediaType + "] " + m.MediaCaption
		} else if text == "" && m.MediaType != "" {
			text = "[" + m.MediaType + "]"
		}
		if m.Snippet != "" {
			text = m.Snippet
		}
		if len(text) > 80 {
			text = text[:77] + "..."
		}
		fmt.Fprintf(w, "%d\t%d\t%d\t%s\t%s\n",
			m.ChatID, m.MsgID, m.SenderID,
			m.Timestamp.Format("2006-01-02 15:04"),
			text)
	}
	w.Flush()
}
