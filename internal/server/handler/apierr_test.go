package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"golang.org/x/oauth2"
	"google.golang.org/api/googleapi"
)

func TestMapAPIError_GoogleAPIHTTPStatuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		code     int
		message  string
		wantSubs []string // substrings that must appear in tool error text
	}{
		{
			name:     "401",
			code:     http.StatusUnauthorized,
			message:  "unauthorized",
			wantSubs: []string{"sheets-mcp auth", "authentication"},
		},
		{
			name:     "403",
			code:     http.StatusForbidden,
			message:  "forbidden",
			wantSubs: []string{"403", "Share"},
		},
		{
			name:     "404",
			code:     http.StatusNotFound,
			message:  "not found",
			wantSubs: []string{"404", "spreadsheet"},
		},
		{
			name:     "400 parse range",
			code:     http.StatusBadRequest,
			message:  "Unable to parse range: Sheet1!X",
			wantSubs: []string{"400", "range"},
		},
		{
			name:     "429",
			code:     http.StatusTooManyRequests,
			message:  "rate",
			wantSubs: []string{"429", "rate"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := &googleapi.Error{Code: tt.code, Message: tt.message}
			err := fmt.Errorf("wrapped: %w", g)
			res, ok := mapAPIError(err)
			if !ok || res == nil {
				t.Fatalf("mapAPIError: ok=%v res=%v", ok, res)
			}
			text := textContentString(t, res)
			for _, sub := range tt.wantSubs {
				if !strings.Contains(text, sub) {
					t.Errorf("tool error text %q should contain %q", text, sub)
				}
			}
		})
	}
}

func TestMapAPIError_WrappedGoogleAPIError(t *testing.T) {
	t.Parallel()
	inner := &googleapi.Error{Code: http.StatusNotFound, Message: "missing"}
	err := fmt.Errorf("read range %q: %w", "Sheet1!A1", inner)
	res, ok := mapAPIError(err)
	if !ok || res == nil {
		t.Fatalf("expected mapped tool error, got ok=%v res=%v", ok, res)
	}
	if text := textContentString(t, res); !strings.Contains(text, "404") {
		t.Errorf("expected 404 in message: %q", text)
	}
}

func TestMapAPIError_NonAPIError(t *testing.T) {
	t.Parallel()
	_, ok := mapAPIError(errors.New("something else"))
	if ok {
		t.Fatal("expected not mapped")
	}
}

func TestMapAPIError_UnknownGoogleAPICode(t *testing.T) {
	t.Parallel()
	g := &googleapi.Error{Code: http.StatusInternalServerError, Message: "oops"}
	_, ok := mapAPIError(g)
	if ok {
		t.Fatal("expected 500 not mapped to tool error")
	}
}

func TestMapAPIError_OAuthRetrieveError(t *testing.T) {
	t.Parallel()
	re := &oauth2.RetrieveError{ErrorCode: "invalid_grant"}
	res, ok := mapAPIError(re)
	if !ok || res == nil {
		t.Fatalf("expected mapped tool error, got ok=%v", ok)
	}
	text := textContentString(t, res)
	if !strings.Contains(text, "sheets-mcp auth") {
		t.Errorf("expected auth hint: %q", text)
	}
}

func textContentString(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("no content")
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	return tc.Text
}
