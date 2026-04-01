package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/mcp"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
)

const maxAPIErrorDetail = 160

// mapAPIError maps Google API and OAuth transport errors to MCP tool errors.
// It returns (nil, false) if err should be surfaced as a protocol error
// (unknown API codes, context cancellation, non-API errors).
func mapAPIError(err error) (*mcp.CallToolResult, bool) {
	if err == nil {
		return nil, false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, false
	}

	var g *googleapi.Error
	if errors.As(err, &g) {
		return mapGoogleAPIError(g)
	}

	var re *oauth2.RetrieveError
	if errors.As(err, &re) {
		return mcp.NewToolResultError(retrieveErrorUserMessage(re)), true
	}

	return nil, false
}

func mapGoogleAPIError(g *googleapi.Error) (*mcp.CallToolResult, bool) {
	switch g.Code {
	case http.StatusUnauthorized:
		return mcp.NewToolResultError(
			"authentication failed or access token expired. Run 'sheets-mcp auth' to re-authenticate.",
		), true
	case http.StatusForbidden:
		return mcp.NewToolResultError(
			"access denied (403). Share the spreadsheet with the Google account used for sheets-mcp, or check that the Sheets and Drive APIs are enabled for your GCP project.",
		), true
	case http.StatusNotFound:
		return mcp.NewToolResultError(
			"spreadsheet or file not found (404), or this account cannot access it. Verify the spreadsheet ID and sharing.",
		), true
	case http.StatusBadRequest:
		return mcp.NewToolResultError(badRequestMessage(g)), true
	case http.StatusTooManyRequests:
		return mcp.NewToolResultError("Google API rate limit exceeded (429). Wait briefly and retry."), true
	default:
		return nil, false
	}
}

func badRequestMessage(g *googleapi.Error) string {
	msg := strings.TrimSpace(g.Message)
	if msg == "" && len(g.Errors) > 0 {
		msg = strings.TrimSpace(g.Errors[0].Message)
	}
	msg = oneLine(msg)
	if msg == "" {
		return "invalid request to Google Sheets or Drive (400). Check sheet name, A1 range, and spreadsheet ID."
	}
	// Prefer stable phrasing when the API mentions range/sheet issues.
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "unable to parse range"),
		strings.Contains(lower, "invalid range"),
		strings.Contains(lower, "parse error"):
		return fmt.Sprintf("invalid range or sheet reference (400): %s", truncateRunes(msg, maxAPIErrorDetail))
	default:
		return fmt.Sprintf("Google API rejected the request (400): %s", truncateRunes(msg, maxAPIErrorDetail))
	}
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func truncateRunes(s string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(s) <= limit {
		return s
	}
	runes := []rune(s)
	if len(runes) > limit {
		if limit <= 3 {
			return string(runes[:limit])
		}
		return string(runes[:limit-3]) + "..."
	}
	return s
}

func retrieveErrorUserMessage(re *oauth2.RetrieveError) string {
	if re == nil {
		return "OAuth token error. Run 'sheets-mcp auth' to re-authenticate."
	}
	switch re.ErrorCode {
	case "invalid_grant", "invalid_client":
		return "authentication failed or token was revoked. Run 'sheets-mcp auth' to re-authenticate."
	default:
		return "OAuth token refresh failed. Run 'sheets-mcp auth' to re-authenticate."
	}
}
