package dataexport

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"testing"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

const (
	mimeCSV  = "text/csv"
	mimeXLSX = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
)

// utf8BOM is written at the start of every CSV export so Excel opens it
// as UTF-8.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// exportResult mirrors DataExportResult.
type exportResult struct {
	Filename      string `json:"filename"`
	MimeType      string `json:"mimeType"`
	ContentBase64 string `json:"contentBase64"`
	RowCount      int    `json:"rowCount"`
}

const exportDataTableQuery = `
	query($projectId: ID, $table: DataExportTable!, $format: DataExportFormat!, $scopeId: ID) {
		exportDataTable(projectId: $projectId, table: $table, format: $format, scopeId: $scopeId) {
			filename mimeType contentBase64 rowCount
		}
	}`

const exportDataWorkbookQuery = `
	query($projectId: ID!, $tables: [DataExportTable!]) {
		exportDataWorkbook(projectId: $projectId, tables: $tables) {
			filename mimeType contentBase64 rowCount
		}
	}`

// exportTable calls Query.exportDataTable. Empty projectID / scopeID are
// sent as null.
func exportTable(ctx context.Context, client *graphql.Client, projectID, table, format, scopeID string) (*exportResult, error) {
	vars := map[string]interface{}{"table": table, "format": format, "projectId": nil, "scopeId": nil}
	if projectID != "" {
		vars["projectId"] = projectID
	}
	if scopeID != "" {
		vars["scopeId"] = scopeID
	}
	var resp struct {
		ExportDataTable exportResult `json:"exportDataTable"`
	}
	if err := client.Query(ctx, exportDataTableQuery, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.ExportDataTable, nil
}

// exportWorkbook calls Query.exportDataWorkbook. A nil tables slice is
// sent as null (server default table set).
func exportWorkbook(ctx context.Context, client *graphql.Client, projectID string, tables []string) (*exportResult, error) {
	vars := map[string]interface{}{"projectId": projectID, "tables": nil}
	if tables != nil {
		vars["tables"] = tables
	}
	var resp struct {
		ExportDataWorkbook exportResult `json:"exportDataWorkbook"`
	}
	if err := client.Query(ctx, exportDataWorkbookQuery, vars, &resp); err != nil {
		return nil, err
	}
	return &resp.ExportDataWorkbook, nil
}

func decodeContent(t *testing.T, res *exportResult) []byte {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(res.ContentBase64)
	require.NoError(t, err, "contentBase64 must be standard base64")
	require.NotEmpty(t, raw, "export content must not be empty")
	return raw
}

// parseCSV checks the CSV framing contract (UTF-8 BOM, CRLF line endings)
// and returns all records including the header row.
func parseCSV(t *testing.T, res *exportResult) [][]string {
	t.Helper()
	raw := decodeContent(t, res)
	require.True(t, bytes.HasPrefix(raw, utf8BOM), "CSV must start with a UTF-8 BOM")
	body := raw[len(utf8BOM):]
	require.Contains(t, string(body), "\r\n", "CSV rows must end with CRLF")
	// The server writes with encoding/csv UseCRLF, which also writes an LF
	// inside a quoted field as CRLF (the reader turns it back into LF), so
	// a bare LF anywhere means the line-ending contract is broken.
	require.NotContains(t, string(bytes.ReplaceAll(body, []byte("\r\n"), nil)), "\n",
		"CSV must not contain bare LF line endings")

	r := csv.NewReader(bytes.NewReader(body))
	records, err := r.ReadAll()
	require.NoError(t, err, "CSV must parse (all rows the same width)")
	require.NotEmpty(t, records, "CSV must have at least a header row")
	return records
}

// workbook is a parsed XLSX file: sheet names in order and each sheet's
// rows, padded to the header width so blank trailing cells are "".
type workbook struct {
	Sheets []string
	Rows   map[string][][]string
}

func parseXLSX(t *testing.T, res *exportResult) *workbook {
	t.Helper()
	raw := decodeContent(t, res)
	f, err := excelize.OpenReader(bytes.NewReader(raw))
	require.NoError(t, err, "XLSX must open with excelize")
	defer func() { _ = f.Close() }()

	wb := &workbook{Sheets: f.GetSheetList(), Rows: map[string][][]string{}}
	for _, name := range wb.Sheets {
		rows, err := f.GetRows(name)
		require.NoError(t, err, "read sheet %q", name)
		width := 0
		if len(rows) > 0 {
			width = len(rows[0])
		}
		for i, row := range rows {
			for len(row) < width {
				row = append(row, "")
			}
			rows[i] = row
		}
		wb.Rows[name] = rows
	}
	return wb
}

// column returns the index of header in the header row, failing if absent.
func column(t *testing.T, header []string, name string) int {
	t.Helper()
	for i, h := range header {
		if h == name {
			return i
		}
	}
	require.Failf(t, "missing column", "column %q not in %v", name, header)
	return -1
}

// rowsBy indexes data rows (header excluded) by the value in column col.
func rowsBy(t *testing.T, records [][]string, col string) map[string][]string {
	t.Helper()
	idx := column(t, records[0], col)
	out := make(map[string][]string, len(records)-1)
	for _, r := range records[1:] {
		out[r[idx]] = r
	}
	return out
}

// cell returns the value of column col in row, looked up via header.
func cell(t *testing.T, header, row []string, col string) string {
	t.Helper()
	return row[column(t, header, col)]
}
