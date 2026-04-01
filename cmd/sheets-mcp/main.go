// Package main is the main package for the sheets-mcp application.
package main

import (
	"fmt"
	"strings"

	"github.com/mab-go/sheets-mcp/internal/auth"
	"github.com/mab-go/sheets-mcp/internal/logging"
	"github.com/mab-go/sheets-mcp/internal/server"
	"github.com/mab-go/sheets-mcp/internal/version"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	cmd = &cobra.Command{
		Use:     "sheets-mcp",
		Short:   "Google Sheets MCP Server",
		Long:    "An MCP server for reading and writing Google Sheets data via the Sheets API.",
		Version: fmt.Sprintf("sheets-mcp %s (%s; %s)", version.Version, version.ShortCommit(), version.Date),
	}

	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server (stdio transport)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return server.RunStdioServer()
		},
	}

	authCmd = &cobra.Command{
		Use:   "auth",
		Short: "Manage OAuth authentication",
		RunE:  runAuth,
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information and exit",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("sheets-mcp %s (%s; %s)\n", version.Version, version.ShortCommit(), version.Date)
		},
	}
)

func runAuth(cmd *cobra.Command, _ []string) error {
	status, _ := cmd.Flags().GetBool("status")
	revoke, _ := cmd.Flags().GetBool("revoke")

	switch {
	case status:
		return runAuthStatus()
	case revoke:
		return runAuthRevoke()
	default:
		return runAuthFlow()
	}
}

func runAuthStatus() error {
	tok, err := auth.LoadToken()
	if err != nil {
		fmt.Printf("Not authenticated: %v\n", err)
		return nil
	}

	if auth.TokenValid(tok) {
		fmt.Printf("Authenticated.\n")
		fmt.Printf("  Access token expires: %s\n", tok.Expiry.Local().Format("2006-01-02 15:04:05"))
		if tok.RefreshToken != "" {
			fmt.Printf("  Refresh token: present\n")
		}
	} else {
		fmt.Printf("Token expired at %s.\n", tok.Expiry.Local().Format("2006-01-02 15:04:05"))
		if tok.RefreshToken != "" {
			fmt.Printf("  Refresh token present — will refresh automatically on next use.\n")
		} else {
			fmt.Printf("  No refresh token — run 'sheets-mcp auth' to re-authenticate.\n")
		}
	}

	return nil
}

func runAuthRevoke() error {
	tok, err := auth.LoadToken()
	if err != nil {
		fmt.Printf("No token to revoke: %v\n", err)
		return nil
	}

	if err := auth.RevokeToken(tok); err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}

	fmt.Printf("Token revoked and deleted.\n")
	return nil
}

func runAuthFlow() error {
	cfg, err := auth.LoadConfig()
	if err != nil {
		return err
	}

	tok, err := auth.RunOAuthFlow(cfg)
	if err != nil {
		return err
	}

	fmt.Printf("Authentication successful! Token expires %s.\n",
		tok.Expiry.Local().Format("2006-01-02 15:04:05"))
	return nil
}

// init registers cobra/viper setup hooks, normalizes global flag names (underscores to
// hyphens), and configures the version output template.
func init() {
	cobra.OnInitialize(func() {
		viper.SetEnvPrefix("mcp_sheets")
		viper.AutomaticEnv()
	})
	cmd.SetGlobalNormalizationFunc(wordSepNormalizeFunc)
	cmd.SetVersionTemplate("{{.Version}}\n")

	authCmd.Flags().Bool("status", false, "Check token state without triggering a flow")
	authCmd.Flags().Bool("revoke", false, "Revoke and delete stored token")

	cmd.AddCommand(serveCmd)
	cmd.AddCommand(authCmd)
	cmd.AddCommand(versionCmd)
}

// wordSepNormalizeFunc normalizes flag names by replacing underscores with hyphens.
func wordSepNormalizeFunc(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	name = strings.ReplaceAll(name, "_", "-")
	return pflag.NormalizedName(name)
}

// main runs the root cobra command; on failure it logs a fatal error and exits.
func main() {
	if err := cmd.Execute(); err != nil {
		logging.WithError(err).Fatal("Failed to execute root command")
	}
}
