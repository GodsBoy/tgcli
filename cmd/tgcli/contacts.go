package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/GodsBoy/tgcli/internal/format"
)

func newContactsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contacts",
		Short: "Manage contacts",
	}
	cmd.AddCommand(newContactsListCmd(flags))
	return cmd
}

func newContactsListCmd(flags *rootFlags) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all contacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := newAppContext(cmd.Context(), flags, false)
			if err != nil {
				return err
			}
			defer ac.close()

			contacts, err := ac.db.ListContacts(limit)
			if err != nil {
				return fmt.Errorf("list contacts: %w", err)
			}

			if flags.asJSON {
				return format.WriteJSON(os.Stdout, contacts)
			}

			if len(contacts) == 0 {
				fmt.Println("No contacts found. Run 'tgcli sync' first.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "USER_ID\tNAME\tUSERNAME\tPHONE")
			for _, c := range contacts {
				name := c.FirstName
				if c.LastName != "" {
					name += " " + c.LastName
				}
				fmt.Fprintf(w, "%d\t%s\t@%s\t%s\n",
					c.UserID, name, c.Username, c.Phone)
			}
			w.Flush()
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 500, "max number of contacts")
	return cmd
}
