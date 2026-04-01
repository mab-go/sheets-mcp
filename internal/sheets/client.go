package sheets

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mab-go/sheets-mcp/internal/auth"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	sheets "google.golang.org/api/sheets/v4"
)

// defaultHTTPClientTimeout is the maximum time for a single Sheets/Drive HTTP round trip
// (including reading the response body). Per-request cancellation still applies via
// Context(ctx) on API calls; this is a backstop when the caller does not set a deadline.
const defaultHTTPClientTimeout = 90 * time.Second

// batchGetFirstRowRangesPerRequest limits Values.BatchGet ranges per HTTP GET;
// batchGet is GET with repeated query params, so URL length is the practical cap.
const batchGetFirstRowRangesPerRequest = 40

// spreadsheetMetadataFields limits Spreadsheets.Get to properties needed for sheets_info.
const spreadsheetMetadataFields = "spreadsheetId,properties.title,sheets.properties(sheetId,title,index,gridProperties(rowCount,columnCount))"

// Client wraps authenticated Google Sheets and Drive API services.
type Client struct {
	sheets *sheets.Service
	drive  *drive.Service
}

// NewClient creates a Client with automatic token refresh. The oauth2 token
// source handles refreshing transparently on each API call.
func NewClient(cfg *auth.Config, tok *oauth2.Token) (*Client, error) {
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes:       auth.Scopes,
		Endpoint:     google.Endpoint,
	}

	ctx := context.Background()
	httpClient := oauthCfg.Client(ctx, tok)
	httpClient.Timeout = defaultHTTPClientTimeout

	sheetsSvc, err := sheets.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Sheets service: %w", err)
	}

	driveSvc, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create Drive service: %w", err)
	}

	return &Client{sheets: sheetsSvc, drive: driveSvc}, nil
}

// GetSpreadsheetMetadata returns sheet names, dimensions, and the first row of
// each sheet for the given spreadsheet.
func (c *Client) GetSpreadsheetMetadata(ctx context.Context, spreadsheetID string) (*SpreadsheetMetadata, error) {
	resp, err := c.sheets.Spreadsheets.Get(spreadsheetID).
		IncludeGridData(false).
		Fields(googleapi.Field(spreadsheetMetadataFields)).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("get spreadsheet metadata: %w", err)
	}

	meta := &SpreadsheetMetadata{
		SpreadsheetID: resp.SpreadsheetId,
		Title:         resp.Properties.Title,
	}

	sheetProps := resp.Sheets
	meta.Sheets = make([]SheetInfo, 0, len(sheetProps))
	for _, s := range sheetProps {
		props := s.Properties
		info := SheetInfo{Title: props.Title, Index: int(props.Index)}
		if props.GridProperties != nil {
			info.RowCount = int(props.GridProperties.RowCount)
			info.ColCount = int(props.GridProperties.ColumnCount)
		}
		meta.Sheets = append(meta.Sheets, info)
	}

	for start := 0; start < len(meta.Sheets); start += batchGetFirstRowRangesPerRequest {
		end := start + batchGetFirstRowRangesPerRequest
		if end > len(meta.Sheets) {
			end = len(meta.Sheets)
		}
		ranges := make([]string, 0, end-start)
		for i := start; i < end; i++ {
			r := QuoteSheetNameForA1(meta.Sheets[i].Title) + "!1:1"
			ranges = append(ranges, r)
		}

		batchResp, err := c.sheets.Spreadsheets.Values.BatchGet(spreadsheetID).
			Ranges(ranges...).
			ValueRenderOption("UNFORMATTED_VALUE").
			DateTimeRenderOption("FORMATTED_STRING").
			Context(ctx).
			Do()
		if err != nil {
			return nil, fmt.Errorf("batch get first rows [%d:%d]: %w", start, end, err)
		}

		for j, vr := range batchResp.ValueRanges {
			idx := start + j
			if idx >= len(meta.Sheets) {
				break
			}
			if len(vr.Values) == 0 {
				continue
			}
			rows := convertValues(vr.Values)
			meta.Sheets[idx].FirstRow = rows[0]
		}
	}

	return meta, nil
}

// ReadRange reads cell data from the specified range. The range should be in
// A1 notation, optionally prefixed with a sheet name (e.g. "Sheet1!A1:D10").
func (c *Client) ReadRange(ctx context.Context, spreadsheetID, sheetRange string) (*RangeData, error) {
	resp, err := c.sheets.Spreadsheets.Values.Get(spreadsheetID, sheetRange).
		ValueRenderOption("UNFORMATTED_VALUE").
		DateTimeRenderOption("FORMATTED_STRING").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("read range %q: %w", sheetRange, err)
	}

	return &RangeData{
		Range:  resp.Range,
		Values: convertValues(resp.Values),
	}, nil
}

// WriteRange writes values to the specified range, overwriting existing data.
func (c *Client) WriteRange(ctx context.Context, spreadsheetID, sheetRange string, values [][]any) (*WriteResult, error) {
	vr := &sheets.ValueRange{Values: toInterfaceSlice(values)}

	resp, err := c.sheets.Spreadsheets.Values.Update(spreadsheetID, sheetRange, vr).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("write range %q: %w", sheetRange, err)
	}

	return &WriteResult{
		UpdatedRange: resp.UpdatedRange,
		UpdatedRows:  int(resp.UpdatedRows),
		UpdatedCols:  int(resp.UpdatedColumns),
		UpdatedCells: int(resp.UpdatedCells),
	}, nil
}

// AppendRows appends rows after the last row of existing data in the sheet.
func (c *Client) AppendRows(ctx context.Context, spreadsheetID, sheet string, values [][]any) (*WriteResult, error) {
	// Use the sheet name as the range — the API appends after existing data.
	vr := &sheets.ValueRange{Values: toInterfaceSlice(values)}

	quoted := QuoteSheetNameForA1(sheet)
	resp, err := c.sheets.Spreadsheets.Values.Append(spreadsheetID, quoted, vr).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("append rows to %q: %w", sheet, err)
	}

	upd := resp.Updates
	return &WriteResult{
		UpdatedRange: upd.UpdatedRange,
		UpdatedRows:  int(upd.UpdatedRows),
		UpdatedCols:  int(upd.UpdatedColumns),
		UpdatedCells: int(upd.UpdatedCells),
	}, nil
}

// ClearRange clears all values in the specified range without deleting cells.
func (c *Client) ClearRange(ctx context.Context, spreadsheetID, sheetRange string) (string, error) {
	resp, err := c.sheets.Spreadsheets.Values.Clear(spreadsheetID, sheetRange, &sheets.ClearValuesRequest{}).
		Context(ctx).
		Do()
	if err != nil {
		return "", fmt.Errorf("clear range %q: %w", sheetRange, err)
	}

	return resp.ClearedRange, nil
}

// FindSpreadsheets searches Drive for Google Sheets files matching the query.
func (c *Client) FindSpreadsheets(ctx context.Context, query string) ([]SearchResult, error) {
	driveQuery := fmt.Sprintf(
		"mimeType='application/vnd.google-apps.spreadsheet' and name contains '%s' and trashed=false",
		escapeDriveQuery(query),
	)

	resp, err := c.drive.Files.List().
		Q(driveQuery).
		Fields("files(id, name, modifiedTime)").
		OrderBy("modifiedTime desc").
		PageSize(20).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("search spreadsheets: %w", err)
	}

	results := make([]SearchResult, 0, len(resp.Files))
	for _, f := range resp.Files {
		r := SearchResult{
			SpreadsheetID: f.Id,
			Title:         f.Name,
		}
		if t, err := parseTime(f.ModifiedTime); err == nil {
			r.ModifiedTime = t
		}
		results = append(results, r)
	}

	return results, nil
}

// convertValues converts the Google API's [][]interface{} to our [][]any.
// The Google API returns float64 for numbers, string for strings, and bool
// for booleans.
func convertValues(rows [][]interface{}) [][]any {
	result := make([][]any, len(rows))
	copy(result, rows)
	return result
}

// toInterfaceSlice converts [][]any to [][]interface{} for the Google API.
func toInterfaceSlice(rows [][]any) [][]interface{} {
	result := make([][]interface{}, len(rows))
	copy(result, rows)
	return result
}

// escapeDriveQuery escapes backslashes and single quotes in a Drive query string value.
func escapeDriveQuery(s string) string {
	// Escape backslashes first to avoid double-escaping the backslashes added for single quotes.
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `\'`)
	return s
}

// parseTime parses a time string from the Drive API (RFC 3339).
func parseTime(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
