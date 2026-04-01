// Package server provides the MCP server for sheets-mcp.
package server

import (
	"context"
	"fmt"
	"io"

	"github.com/mab-go/sheets-mcp/internal/auth"
	"github.com/mab-go/sheets-mcp/internal/logging"
	"github.com/mab-go/sheets-mcp/internal/server/handler"
	"github.com/mab-go/sheets-mcp/internal/sheets"
	"github.com/mab-go/sheets-mcp/internal/version"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func newSheetsServer(ctx context.Context, h *handler.SheetsHandler) *mcpserver.MCPServer {
	log, _ := logging.FromContext(ctx)

	hooks := &mcpserver.Hooks{}
	hooks.AddBeforeInitialize(hookAddBeforeInitialize(log))
	hooks.AddAfterInitialize(hookAddAfterInitialize(log))

	s := mcpserver.NewMCPServer(
		"sheets-mcp",
		version.Version,
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithLogging(),
		mcpserver.WithHooks(hooks),
	)

	s.AddTool(toolSheetsFind, h.SheetsFind)
	s.AddTool(toolSheetsInfo, h.SheetsInfo)
	s.AddTool(toolSheetsGet, h.SheetsGet)
	s.AddTool(toolSheetsUpdate, h.SheetsUpdate)
	s.AddTool(toolSheetsAppend, h.SheetsAppend)
	s.AddTool(toolSheetsLookup, h.SheetsLookup)
	s.AddTool(toolSheetsClear, h.SheetsClear)

	return s
}

// RunStdioServer loads config and token, creates the Sheets client, and starts
// the MCP server on stdio. If no token exists, the OAuth flow is triggered
// automatically.
func RunStdioServer() error {
	log := logging.NewDefaultLogger()
	ctx := logging.NewContext(context.Background(), log)

	cfg, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	tok, err := auth.LoadToken()
	if err != nil {
		log.Info("No stored token found — starting OAuth flow")
		tok, err = auth.RunOAuthFlow(cfg)
		if err != nil {
			return fmt.Errorf("OAuth flow: %w", err)
		}
	}

	oauthCfg := auth.OAuthConfig(cfg, "")
	tok, err = auth.RefreshIfNeeded(oauthCfg, tok)
	if err != nil {
		log.Info("Token refresh failed — starting OAuth flow")
		tok, err = auth.RunOAuthFlow(cfg)
		if err != nil {
			return fmt.Errorf("OAuth flow: %w", err)
		}
	}

	client, err := sheets.NewClient(cfg, tok)
	if err != nil {
		return fmt.Errorf("create Sheets client: %w", err)
	}

	h := handler.NewSheetsHandler(client, cfg)

	shutdown := func() {
		log.Info("Server shutdown complete")
	}
	defer shutdown()

	srv := newSheetsServer(ctx, h)

	if err := mcpserver.ServeStdio(srv); err != nil {
		if err != context.Canceled && err != io.EOF {
			log.WithError(err).Error("Error running MCP server")
			return fmt.Errorf("failed to start stdio MCP server: %w", err)
		}
	}

	log.Info("Received shutdown signal; shutting down server...")

	return nil
}
