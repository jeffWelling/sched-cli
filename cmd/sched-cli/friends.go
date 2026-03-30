package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/jeff/sched-cli/internal/app"
	"github.com/spf13/cobra"
)

var friendsCmd = &cobra.Command{
	Use:   "friends",
	Short: "Manage friend list",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var friendsAddCmd = &cobra.Command{
	Use:   "add NICKNAME USERNAME",
	Short: "Add a friend (nickname maps to Sched username)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "friends-add")
		if err != nil {
			return err
		}
		defer a.Close()

		nickname := args[0]
		username := args[1]

		if err := a.Store().AddFriend(nickname, username); err != nil {
			return fmt.Errorf("adding friend: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Added friend '%s' -> %s\n", nickname, username)
		return nil
	},
}

var friendsRemoveCmd = &cobra.Command{
	Use:   "remove NICKNAME",
	Short: "Remove a friend by nickname",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "friends-remove")
		if err != nil {
			return err
		}
		defer a.Close()

		nickname := args[0]

		err = a.Store().RemoveFriend(nickname)
		if err == sql.ErrNoRows {
			return fmt.Errorf("friend '%s' not found", nickname)
		}
		if err != nil {
			return fmt.Errorf("removing friend: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Removed friend '%s'\n", nickname)
		return nil
	},
}

var friendsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all friends",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "friends-list")
		if err != nil {
			return err
		}
		defer a.Close()

		friends, err := a.Store().ListFriends()
		if err != nil {
			return fmt.Errorf("listing friends: %w", err)
		}

		return a.Output().FormatFriends(friends)
	},
}

func init() {
	rootCmd.AddCommand(friendsCmd)
	friendsCmd.AddCommand(friendsAddCmd, friendsRemoveCmd, friendsListCmd)
}
