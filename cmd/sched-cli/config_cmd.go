package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jeff/sched-cli/internal/app"
	"github.com/jeff/sched-cli/internal/auth"
	"github.com/jeff/sched-cli/internal/output"
	"github.com/jeff/sched-cli/internal/paths"
	"github.com/spf13/cobra"
)

var (
	initUsername         string
	initPassword         string
	initCredentialsStdin bool
	initToken            string
	initBrowser          bool
	initFromBrowser      bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Setup and configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize sched-cli configuration",
	Long:  "Set up authentication, event URL, and initial sync.",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "config-init")
		if err != nil {
			return err
		}
		defer a.Close()

		authenticator := auth.New(output.IsTerminal(os.Stdout.Fd()))
		var result *auth.AuthResult

		switch {
		case initUsername != "" || initPassword != "":
			if initUsername == "" || initPassword == "" {
				return fmt.Errorf("both --username and --password are required")
			}
			result, err = authenticator.LoginWithCredentials(initUsername, initPassword)

		case initCredentialsStdin:
			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() {
				return fmt.Errorf("expected email on first line of stdin")
			}
			email := strings.TrimSpace(scanner.Text())
			if !scanner.Scan() {
				return fmt.Errorf("expected password on second line of stdin")
			}
			password := strings.TrimSpace(scanner.Text())
			result, err = authenticator.LoginWithCredentials(email, password)

		case initToken != "":
			result, err = authenticator.LoginWithToken(initToken)

		case initBrowser:
			result, err = authenticator.LoginWithBrowser(5 * time.Minute)

		case initFromBrowser:
			result, err = authenticator.LoginFromFirefox()

		default:
			if !authenticator.IsInteractive() {
				return fmt.Errorf("no auth flags provided and stdin is not a terminal.\n\nUsage:\n  sched-cli config init --username EMAIL --password PASS\n  sched-cli config init --credentials-from-stdin\n  sched-cli config init --token TOKEN\n  sched-cli config init --browser\n  sched-cli config init --from-browser")
			}
			result, err = interactiveAuth(authenticator)
		}

		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		// Update config with auth result.
		cfg := a.Config()
		cfg.Token = result.Token
		cfg.UContext = result.UContext
		cfg.AuthMethod = result.Method

		// Prompt for event URL if interactive and not already set.
		if authenticator.IsInteractive() && cfg.EventURL == "" {
			fmt.Print("Enter event URL (e.g., https://myconf2026.sched.com): ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				url := strings.TrimSpace(scanner.Text())
				if url != "" {
					cfg.EventURL = url
				}
			}
		}

		if err := a.SaveConfig(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Fprintln(os.Stderr, "Configuration saved. Run 'sched-cli sync' to pull session data.")
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "config-show")
		if err != nil {
			return err
		}
		defer a.Close()

		cfg := a.Config()
		redactedToken := ""
		if cfg.Token != "" {
			if len(cfg.Token) > 8 {
				redactedToken = cfg.Token[:8] + "..."
			} else {
				redactedToken = "***"
			}
		}

		fmt.Printf("event_url:          %s\n", cfg.EventURL)
		fmt.Printf("username:           %s\n", cfg.Username)
		fmt.Printf("auth_method:        %s\n", cfg.AuthMethod)
		fmt.Printf("token:              %s\n", redactedToken)
		fmt.Printf("directory_style:    %s\n", cfg.DirectoryStyle)
		fmt.Printf("log_retention_days: %d\n", cfg.LogRetentionDays)
		fmt.Printf("cache_ttl_hours:    %d\n", cfg.CacheTTLHours)
		fmt.Printf("syslog:             %t\n", cfg.Syslog)
		fmt.Printf("config_dir:         %s\n", a.Paths().ConfigDir())
		fmt.Printf("cache_dir:          %s\n", a.Paths().CacheDir())
		fmt.Printf("log_dir:            %s\n", a.Paths().LogDir())
		return nil
	},
}

// configSetAllowedKeys lists the keys that can be set via `config set`.
var configSetAllowedKeys = map[string]bool{
	"event_url":          true,
	"username":           true,
	"directory_style":    true,
	"log_retention_days": true,
	"cache_ttl_hours":    true,
	"syslog":             true,
}

// configSetSensitiveKeys are keys that require `config login` instead.
var configSetSensitiveKeys = map[string]bool{
	"token":       true,
	"ucontext":    true,
	"auth_method": true,
}

var configSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		if configSetSensitiveKeys[key] {
			return fmt.Errorf("cannot set '%s' directly. Use 'sched-cli config login' to change auth credentials", key)
		}
		if !configSetAllowedKeys[key] {
			keys := make([]string, 0, len(configSetAllowedKeys))
			for k := range configSetAllowedKeys {
				keys = append(keys, k)
			}
			return fmt.Errorf("unknown config key '%s'. Allowed keys: %s", key, strings.Join(keys, ", "))
		}

		a, err := app.New(debug, jsonFlag, jsonPretty, "config-set")
		if err != nil {
			return err
		}
		defer a.Close()

		cfg := a.Config()
		switch key {
		case "event_url":
			cfg.EventURL = value
		case "username":
			cfg.Username = value
		case "directory_style":
			switch paths.DirectoryStyle(value) {
			case paths.StylePlatform, paths.StyleXDG:
				cfg.DirectoryStyle = paths.DirectoryStyle(value)
			default:
				return fmt.Errorf("directory_style must be 'platform' or 'xdg'")
			}
		case "log_retention_days":
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("log_retention_days must be an integer: %w", err)
			}
			cfg.LogRetentionDays = n
		case "cache_ttl_hours":
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("cache_ttl_hours must be an integer: %w", err)
			}
			cfg.CacheTTLHours = n
		case "syslog":
			b, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("syslog must be true or false: %w", err)
			}
			cfg.Syslog = b
		}

		if err := a.SaveConfig(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Set %s = %s\n", key, value)
		return nil
	},
}

var configLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Re-authenticate with Sched.com",
	Long:  "Change authentication credentials. Always shows the full auth method menu.",
	RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.New(debug, jsonFlag, jsonPretty, "config-login")
		if err != nil {
			return err
		}
		defer a.Close()

		authenticator := auth.New(output.IsTerminal(os.Stdout.Fd()))

		var result *auth.AuthResult

		switch {
		case initUsername != "" || initPassword != "":
			if initUsername == "" || initPassword == "" {
				return fmt.Errorf("both --username and --password are required")
			}
			result, err = authenticator.LoginWithCredentials(initUsername, initPassword)

		case initToken != "":
			result, err = authenticator.LoginWithToken(initToken)

		case initBrowser:
			result, err = authenticator.LoginWithBrowser(5 * time.Minute)

		case initFromBrowser:
			result, err = authenticator.LoginFromFirefox()

		default:
			if !authenticator.IsInteractive() {
				return fmt.Errorf("no auth flags provided and stdin is not a terminal")
			}
			result, err = interactiveAuth(authenticator)
		}

		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		cfg := a.Config()
		cfg.Token = result.Token
		cfg.UContext = result.UContext
		cfg.AuthMethod = result.Method

		if err := a.SaveConfig(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Fprintln(os.Stderr, "Authentication updated.")
		return nil
	},
}

func interactiveAuth(authenticator *auth.Authenticator) (*auth.AuthResult, error) {
	fmt.Println("Choose authentication method:")
	fmt.Println("  1. Email + password")
	fmt.Println("  2. Paste token")
	fmt.Println("  3. Browser (loopback)")
	fmt.Println("  4. Firefox cookies")
	fmt.Print("Enter choice [1-4]: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil, fmt.Errorf("no input received")
	}
	choice := strings.TrimSpace(scanner.Text())

	switch choice {
	case "1":
		fmt.Print("Email: ")
		if !scanner.Scan() {
			return nil, fmt.Errorf("no email input")
		}
		email := strings.TrimSpace(scanner.Text())
		fmt.Print("Password: ")
		if !scanner.Scan() {
			return nil, fmt.Errorf("no password input")
		}
		password := strings.TrimSpace(scanner.Text())
		return authenticator.LoginWithCredentials(email, password)

	case "2":
		fmt.Print("Paste token: ")
		if !scanner.Scan() {
			return nil, fmt.Errorf("no token input")
		}
		token := strings.TrimSpace(scanner.Text())
		return authenticator.LoginWithToken(token)

	case "3":
		return authenticator.LoginWithBrowser(5 * time.Minute)

	case "4":
		return authenticator.LoginFromFirefox()

	default:
		return nil, fmt.Errorf("invalid choice: %s", choice)
	}
}

func init() {
	configInitCmd.Flags().StringVar(&initUsername, "username", "", "Sched.com email address")
	configInitCmd.Flags().StringVar(&initPassword, "password", "", "Sched.com password")
	configInitCmd.Flags().BoolVar(&initCredentialsStdin, "credentials-from-stdin", false, "Read email and password from stdin (one per line)")
	configInitCmd.Flags().StringVar(&initToken, "token", "", "Auth token value")
	configInitCmd.Flags().BoolVar(&initBrowser, "browser", false, "Authenticate via browser loopback")
	configInitCmd.Flags().BoolVar(&initFromBrowser, "from-browser", false, "Extract cookies from Firefox")

	configLoginCmd.Flags().StringVar(&initUsername, "username", "", "Sched.com email address")
	configLoginCmd.Flags().StringVar(&initPassword, "password", "", "Sched.com password")
	configLoginCmd.Flags().StringVar(&initToken, "token", "", "Auth token value")
	configLoginCmd.Flags().BoolVar(&initBrowser, "browser", false, "Authenticate via browser loopback")
	configLoginCmd.Flags().BoolVar(&initFromBrowser, "from-browser", false, "Extract cookies from Firefox")

	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd, configShowCmd, configSetCmd, configLoginCmd)
}
