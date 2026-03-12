package main

import (
	"database/sql"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/GodsBoy/tgcli/internal/format"
)

func newGroupsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groups",
		Short: "Manage groups",
	}
	cmd.AddCommand(newGroupsListCmd(flags))
	cmd.AddCommand(newGroupsInfoCmd(flags))
	return cmd
}

func newGroupsListCmd(flags *rootFlags) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := newAppContext(cmd.Context(), flags, false)
			if err != nil {
				return err
			}
			defer ac.close()

			groups, err := ac.db.ListGroups(limit)
			if err != nil {
				return fmt.Errorf("list groups: %w", err)
			}

			if flags.asJSON {
				return format.WriteJSON(os.Stdout, groups)
			}

			if len(groups) == 0 {
				fmt.Println("No groups found. Run 'tgcli sync' first.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "CHAT_ID\tTITLE\tMEMBERS\tCREATED")
			for _, g := range groups {
				fmt.Fprintf(w, "%d\t%s\t%d\t%s\n",
					g.ChatID, g.Title, g.MemberCount,
					g.CreatedTS.Format("2006-01-02"))
			}
			w.Flush()
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 200, "max number of groups")
	return cmd
}

func newGroupsInfoCmd(flags *rootFlags) *cobra.Command {
	var chatID int64

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show group details and members",
		RunE: func(cmd *cobra.Command, args []string) error {
			if chatID == 0 {
				return fmt.Errorf("--chat is required")
			}

			ac, err := newAppContext(cmd.Context(), flags, false)
			if err != nil {
				return err
			}
			defer ac.close()

			group, err := ac.db.GetGroup(chatID)
			if err == sql.ErrNoRows {
				return fmt.Errorf("group not found: %d", chatID)
			}
			if err != nil {
				return fmt.Errorf("get group: %w", err)
			}

			participants, err := ac.db.GetGroupParticipants(chatID)
			if err != nil {
				return fmt.Errorf("get participants: %w", err)
			}

			if flags.asJSON {
				return format.WriteJSON(os.Stdout, map[string]interface{}{
					"group":        group,
					"participants": participants,
				})
			}

			fmt.Printf("Group:     %s\n", group.Title)
			fmt.Printf("Chat ID:   %d\n", group.ChatID)
			fmt.Printf("Creator:   %d\n", group.CreatorID)
			fmt.Printf("Members:   %d\n", group.MemberCount)
			fmt.Printf("Created:   %s\n", group.CreatedTS.Format("2006-01-02 15:04"))

			if len(participants) > 0 {
				fmt.Printf("\nParticipants:\n")
				w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
				fmt.Fprintln(w, "  USER_ID\tROLE")
				for _, p := range participants {
					fmt.Fprintf(w, "  %d\t%s\n", p.UserID, p.Role)
				}
				w.Flush()
			}
			return nil
		},
	}

	cmd.Flags().Int64Var(&chatID, "chat", 0, "group chat ID")
	return cmd
}
