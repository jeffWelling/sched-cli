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
	Long: `Manage your local friend list. Friends map a short nickname to a Sched
username, so you can use "sched-cli compare --with alice" instead of
remembering full Sched usernames.

Friends are stored locally and not synced to Sched.com.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var friendsAddCmd = &cobra.Command{
	Use:   "add NICKNAME USERNAME",
	Short: "Add a friend (nickname maps to Sched username)",
	Long: `Add a friend by mapping a local nickname to their Sched username. The
nickname is what you use with "sched-cli compare --with"; the username
is their Sched.com profile identifier.

If the nickname already exists, it will be updated with the new username.`,
	Example: `  # Add a friend
  sched-cli friends add alice alice.smith42

  # Add another with a descriptive nickname
  sched-cli friends add bob bob-from-platform-team`,
	Args: cobra.ExactArgs(2),
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
	Long: `Remove a friend from the local friend list by their nickname. Returns an
error if the nickname does not exist.`,
	Example: `  # Remove a friend
  sched-cli friends remove alice`,
	Args: cobra.ExactArgs(1),
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
	Long: `List all friends in the local friend list, showing each nickname and the
Sched username it maps to.`,
	Example: `  # List all friends
  sched-cli friends list

  # Output as JSON
  sched-cli friends list --json`,
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
