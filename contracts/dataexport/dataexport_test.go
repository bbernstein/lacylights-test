package dataexport

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// Contract: the column headers of every exportable table. lacylights-fe
// shows these in its table picker (via dataExportTables) and users build
// spreadsheets around them, so header changes are breaking changes.
var expectedHeaders = map[string][]string{
	"FIXTURE_TYPES":   {"Manufacturer", "Model", "Mode", "Channel count", "Type", "Source", "Fixtures in project using it"},
	"FIXTURE_LIBRARY": {"Manufacturer", "Model", "Mode", "Channel count", "Type", "Source"},
	"FIXTURES": {"Name", "Manufacturer", "Model", "Mode", "Type", "Universe", "Start channel", "End channel",
		"Channel count", "Tags", "Groups", "Description"},
	"LOOKS":        {"Name", "Description", "Fixture count", "Created (UTC)", "Updated (UTC)"},
	"LOOK_DETAILS": {"Fixture", "Manufacturer", "Model", "Mode", "Universe", "Start address"},
	"EFFECTS": {"Name", "Description", "Type", "Waveform", "Frequency (Hz)", "Amplitude (%)", "Offset (%)",
		"Phase offset (deg)", "Easing", "Priority band", "Priority", "Composition", "On cue change",
		"Fade duration (s)", "Master value", "Fixture count"},
	"LOOK_BOARD_ENTRIES": {"Board", "Button label", "Look", "X", "Y", "Width", "Height", "Color", "Board fade time (s)"},
	"CUE_LISTS":          {"Name", "Description", "Cue count", "Loop", "Created (UTC)", "Updated (UTC)"},
	"CUE_LIST_DETAILS": {"Cue #", "Name", "Look", "Fade in (s)", "Fade out (s)", "Follow time (s)", "Easing",
		"Skip", "Notes"},
}

// Contract: table order, labels (also used as XLSX sheet names) and scopes.
var expectedTables = []struct {
	Table, Label, Scope string
}{
	{"FIXTURE_TYPES", "Fixture Types", "project"},
	{"FIXTURE_LIBRARY", "Fixture Library", "none"},
	{"FIXTURES", "Fixtures", "project"},
	{"LOOKS", "Looks", "project"},
	{"LOOK_DETAILS", "Look Details", "look"},
	{"EFFECTS", "Effects", "project"},
	{"LOOK_BOARD_ENTRIES", "Look Board Entries", "lookBoard"},
	{"CUE_LISTS", "Cue Lists", "project"},
	{"CUE_LIST_DETAILS", "Cue List Details", "cueList"},
}

func labelOf(table string) string {
	for _, e := range expectedTables {
		if e.Table == table {
			return e.Label
		}
	}
	return ""
}

var datedFilename = regexp.MustCompile(`-\d{4}-\d{2}-\d{2}\.(csv|xlsx)$`)

func newClient(t *testing.T) (*graphql.Client, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)
	return graphql.NewClient(""), ctx
}

// TestDataExportTables validates the table listing the frontend uses to
// build its picker.
func TestDataExportTables(t *testing.T) {
	client, ctx := newClient(t)

	var resp struct {
		DataExportTables []struct {
			Table   string   `json:"table"`
			Label   string   `json:"label"`
			Scope   string   `json:"scope"`
			Columns []string `json:"columns"`
		} `json:"dataExportTables"`
	}
	require.NoError(t, client.Query(ctx, `query { dataExportTables { table label scope columns } }`, nil, &resp))

	got := resp.DataExportTables
	require.Len(t, got, len(expectedTables), "one entry per DataExportTable enum value")
	for i, want := range expectedTables {
		assert.Equal(t, want.Table, got[i].Table, "table order")
		assert.Equal(t, want.Label, got[i].Label, "label of %s", want.Table)
		assert.Equal(t, want.Scope, got[i].Scope, "scope of %s", want.Table)
		wantCols := expectedHeaders[want.Table]
		if want.Table == "LOOK_DETAILS" {
			// The variable channel run is described by a trailing placeholder.
			wantCols = append(append([]string{}, wantCols...), "Ch 1 … Ch N")
		}
		assert.Equal(t, wantCols, got[i].Columns, "columns of %s", want.Table)
	}
}

// tableCase is one exportDataTable request against the seeded project.
type tableCase struct {
	name      string
	table     string
	projectID string
	scopeID   string
	// wantRows is the expected number of data rows; -1 means "unknown,
	// at least minRows" (the global fixture library).
	wantRows int
	minRows  int
	headers  []string
	// filenamePart must appear in the generated filename.
	filenamePart string
}

func tableCases(s *seededProject) []tableCase {
	lookDetailHeaders := append(append([]string{}, expectedHeaders["LOOK_DETAILS"]...), "Ch 1", "Ch 2", "Ch 3")
	projectSlug := strings.ReplaceAll(s.ProjectName, " ", "-")
	return []tableCase{
		{"FixtureTypes", "FIXTURE_TYPES", s.ProjectID, "", seedFixtureTypes, 0, expectedHeaders["FIXTURE_TYPES"], projectSlug + "-fixture-types"},
		{"FixtureLibraryNoProject", "FIXTURE_LIBRARY", "", "", -1, 2, expectedHeaders["FIXTURE_LIBRARY"], "fixture-library"},
		{"FixtureLibraryWithProject", "FIXTURE_LIBRARY", s.ProjectID, "", -1, 2, expectedHeaders["FIXTURE_LIBRARY"], "fixture-library"},
		{"Fixtures", "FIXTURES", s.ProjectID, "", seedFixtures, 0, expectedHeaders["FIXTURES"], projectSlug + "-fixtures"},
		{"Looks", "LOOKS", s.ProjectID, "", seedLooks, 0, expectedHeaders["LOOKS"], projectSlug + "-looks"},
		{"LookDetails", "LOOK_DETAILS", s.ProjectID, s.LookAID, seedLookADetails, 0, lookDetailHeaders, projectSlug + "-look-details-Look-A"},
		{"Effects", "EFFECTS", s.ProjectID, "", seedEffects, 0, expectedHeaders["EFFECTS"], projectSlug + "-effects"},
		{"LookBoardEntriesAll", "LOOK_BOARD_ENTRIES", s.ProjectID, "", seedBoardEntriesAll, 0, expectedHeaders["LOOK_BOARD_ENTRIES"], projectSlug + "-look-board-entries"},
		{"LookBoardEntriesOneBoard", "LOOK_BOARD_ENTRIES", s.ProjectID, s.MainBoardID, seedBoardEntriesMain, 0, expectedHeaders["LOOK_BOARD_ENTRIES"], projectSlug + "-look-board-entries-Main-Board"},
		{"CueLists", "CUE_LISTS", s.ProjectID, "", seedCueLists, 0, expectedHeaders["CUE_LISTS"], projectSlug + "-cue-lists"},
		{"CueListDetails", "CUE_LIST_DETAILS", s.ProjectID, s.MainCueListID, seedMainCues, 0, expectedHeaders["CUE_LIST_DETAILS"], projectSlug + "-cue-list-details-Main-Show"},
	}
}

func checkRowCount(t *testing.T, tc tableCase, res *exportResult, dataRows int) {
	t.Helper()
	assert.Equal(t, res.RowCount, dataRows, "rowCount must equal the number of data rows in the file")
	if tc.wantRows >= 0 {
		assert.Equal(t, tc.wantRows, res.RowCount, "rowCount vs seeded data")
	} else {
		assert.GreaterOrEqual(t, res.RowCount, tc.minRows, "rowCount vs seeded data")
	}
}

func checkFilename(t *testing.T, tc tableCase, res *exportResult, ext string) {
	t.Helper()
	assert.Contains(t, res.Filename, tc.filenamePart, "filename")
	assert.Regexp(t, datedFilename, res.Filename, "filename ends with -YYYY-MM-DD.<ext>")
	assert.True(t, strings.HasSuffix(res.Filename, "."+ext), "filename extension")
	if tc.table == "FIXTURE_LIBRARY" {
		assert.True(t, strings.HasPrefix(res.Filename, "fixture-library-"),
			"global library filename carries no project name: %s", res.Filename)
	}
}

// TestExportDataTable_CSV exports every table as CSV and checks framing,
// headers, row counts and filename against the seeded project.
func TestExportDataTable_CSV(t *testing.T) {
	client, ctx := newClient(t)
	s := seedProject(t, ctx, client)

	for _, tc := range tableCases(s) {
		t.Run(tc.name, func(t *testing.T) {
			res, err := exportTable(ctx, client, tc.projectID, tc.table, "CSV", tc.scopeID)
			require.NoError(t, err)
			assert.Equal(t, mimeCSV, res.MimeType)
			checkFilename(t, tc, res, "csv")

			records := parseCSV(t, res)
			assert.Equal(t, tc.headers, records[0], "CSV header row")
			checkRowCount(t, tc, res, len(records)-1)
		})
	}
}

// TestExportDataTable_XLSX exports every table as XLSX and checks sheet
// names, headers, row counts and filename against the seeded project.
func TestExportDataTable_XLSX(t *testing.T) {
	client, ctx := newClient(t)
	s := seedProject(t, ctx, client)

	for _, tc := range tableCases(s) {
		t.Run(tc.name, func(t *testing.T) {
			res, err := exportTable(ctx, client, tc.projectID, tc.table, "XLSX", tc.scopeID)
			require.NoError(t, err)
			assert.Equal(t, mimeXLSX, res.MimeType)
			checkFilename(t, tc, res, "xlsx")

			wb := parseXLSX(t, res)
			label := labelOf(tc.table)
			wantSheets := []string{label}
			if tc.table == "LOOK_DETAILS" {
				wantSheets = append(wantSheets, "Channel names")
			}
			require.Equal(t, wantSheets, wb.Sheets, "sheet names")

			rows := wb.Rows[label]
			require.NotEmpty(t, rows, "sheet %q must have a header row", label)
			assert.Equal(t, tc.headers, rows[0], "XLSX header row")
			checkRowCount(t, tc, res, len(rows)-1)
		})
	}
}

// TestExportDataTable_Content spot-checks cell values in each table so
// the exported data (not just its shape) matches what was seeded.
func TestExportDataTable_Content(t *testing.T) {
	client, ctx := newClient(t)
	s := seedProject(t, ctx, client)

	csvOf := func(t *testing.T, table, projectID, scopeID string) [][]string {
		t.Helper()
		res, err := exportTable(ctx, client, projectID, table, "CSV", scopeID)
		require.NoError(t, err)
		return parseCSV(t, res)
	}

	t.Run("FixtureTypes", func(t *testing.T) {
		recs := csvOf(t, "FIXTURE_TYPES", s.ProjectID, "")
		h := recs[0]
		byModel := rowsBy(t, recs, "Model")
		dimmer, rgb := byModel["Dimmer1"], byModel["RGB3"]
		require.NotNil(t, dimmer)
		require.NotNil(t, rgb)
		assert.Equal(t, s.Manufacturer, cell(t, h, dimmer, "Manufacturer"))
		assert.Equal(t, "1", cell(t, h, dimmer, "Channel count"))
		assert.Equal(t, "DIMMER", cell(t, h, dimmer, "Type"))
		assert.Equal(t, "Custom", cell(t, h, dimmer, "Source"))
		assert.Equal(t, "2", cell(t, h, dimmer, "Fixtures in project using it"))
		assert.Equal(t, "3", cell(t, h, rgb, "Channel count"))
		assert.Equal(t, "LED_PAR", cell(t, h, rgb, "Type"))
		assert.Equal(t, "1", cell(t, h, rgb, "Fixtures in project using it"))
	})

	t.Run("FixtureLibrary", func(t *testing.T) {
		recs := csvOf(t, "FIXTURE_LIBRARY", "", "")
		h := recs[0]
		found := 0
		for _, r := range recs[1:] {
			if cell(t, h, r, "Manufacturer") != s.Manufacturer {
				continue
			}
			found++
			assert.Equal(t, "Custom", cell(t, h, r, "Source"))
			switch cell(t, h, r, "Model") {
			case "Dimmer1":
				assert.Equal(t, "1", cell(t, h, r, "Channel count"))
			case "RGB3":
				assert.Equal(t, "3", cell(t, h, r, "Channel count"))
			}
		}
		assert.Equal(t, 2, found, "library lists both seeded definitions (no modes → one row each)")
	})

	t.Run("Fixtures", func(t *testing.T) {
		recs := csvOf(t, "FIXTURES", s.ProjectID, "")
		h := recs[0]
		byName := rowsBy(t, recs, "Name")
		require.Len(t, byName, seedFixtures)
		d1, rgb := byName["Dimmer 1"], byName["RGB Par"]
		require.NotNil(t, d1)
		require.NotNil(t, rgb)
		assert.Equal(t, "1", cell(t, h, d1, "Universe"))
		assert.Equal(t, "1", cell(t, h, d1, "Start channel"))
		assert.Equal(t, "1", cell(t, h, d1, "End channel"))
		assert.Equal(t, "front", cell(t, h, d1, "Tags"))
		assert.Equal(t, "Front Wash", cell(t, h, d1, "Groups"))
		assert.Equal(t, "10", cell(t, h, rgb, "Start channel"))
		assert.Equal(t, "12", cell(t, h, rgb, "End channel"))
		assert.Equal(t, "3", cell(t, h, rgb, "Channel count"))
		assert.Equal(t, "", cell(t, h, rgb, "Groups"), "fixture outside any group has a blank Groups cell")
	})

	t.Run("Looks", func(t *testing.T) {
		recs := csvOf(t, "LOOKS", s.ProjectID, "")
		h := recs[0]
		require.Len(t, recs, 1+seedLooks)
		assert.Equal(t, "Look A", cell(t, h, recs[1], "Name"), "looks sorted by name")
		assert.Equal(t, "2", cell(t, h, recs[1], "Fixture count"))
		assert.Equal(t, "Look B", cell(t, h, recs[2], "Name"))
		assert.Equal(t, "1", cell(t, h, recs[2], "Fixture count"))
		assert.Regexp(t, `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`, cell(t, h, recs[1], "Created (UTC)"))
	})

	t.Run("Effects", func(t *testing.T) {
		recs := csvOf(t, "EFFECTS", s.ProjectID, "")
		h := recs[0]
		byName := rowsBy(t, recs, "Name")
		pulse, hold := byName["Pulse"], byName["Hold"]
		require.NotNil(t, pulse)
		require.NotNil(t, hold)
		assert.Equal(t, "WAVEFORM", cell(t, h, pulse, "Type"))
		assert.Equal(t, "SINE", cell(t, h, pulse, "Waveform"))
		assert.Equal(t, "2", cell(t, h, pulse, "Frequency (Hz)"))
		assert.Equal(t, "80", cell(t, h, pulse, "Amplitude (%)"))
		assert.Equal(t, "1", cell(t, h, pulse, "Fixture count"))
		assert.Equal(t, "STATIC", cell(t, h, hold, "Type"))
		assert.Equal(t, "0", cell(t, h, hold, "Fixture count"))
	})

	t.Run("LookBoardEntries", func(t *testing.T) {
		recs := csvOf(t, "LOOK_BOARD_ENTRIES", s.ProjectID, "")
		h := recs[0]
		require.Len(t, recs, 1+seedBoardEntriesAll)
		// Boards sorted by name; buttons top-to-bottom, left-to-right.
		assert.Equal(t, []string{"Main Board", "Main Board", "Side Board"},
			[]string{cell(t, h, recs[1], "Board"), cell(t, h, recs[2], "Board"), cell(t, h, recs[3], "Board")})
		assert.Equal(t, "A", cell(t, h, recs[1], "Button label"))
		assert.Equal(t, "Look A", cell(t, h, recs[1], "Look"))
		assert.Equal(t, "B", cell(t, h, recs[2], "Button label"))
		assert.Equal(t, "Look B", cell(t, h, recs[2], "Look"))
		assert.Equal(t, "250", cell(t, h, recs[2], "X"))
		assert.Equal(t, "3", cell(t, h, recs[1], "Board fade time (s)"))
		assert.Equal(t, "1.5", cell(t, h, recs[3], "Board fade time (s)"))

		one := csvOf(t, "LOOK_BOARD_ENTRIES", s.ProjectID, s.SideBoardID)
		require.Len(t, one, 2, "scoped to Side Board: header + 1 button")
		assert.Equal(t, "Side A", cell(t, h, one[1], "Button label"))
	})

	t.Run("CueLists", func(t *testing.T) {
		recs := csvOf(t, "CUE_LISTS", s.ProjectID, "")
		h := recs[0]
		require.Len(t, recs, 1+seedCueLists)
		assert.Equal(t, "Empty List", cell(t, h, recs[1], "Name"), "cue lists sorted by name")
		assert.Equal(t, "0", cell(t, h, recs[1], "Cue count"))
		assert.Equal(t, "Main Show", cell(t, h, recs[2], "Name"))
		assert.Equal(t, "3", cell(t, h, recs[2], "Cue count"))
		assert.Equal(t, "No", cell(t, h, recs[2], "Loop"), "booleans are Yes/No")
	})

	t.Run("CueListDetails", func(t *testing.T) {
		recs := csvOf(t, "CUE_LIST_DETAILS", s.ProjectID, s.MainCueListID)
		h := recs[0]
		require.Len(t, recs, 1+seedMainCues)
		assert.Equal(t, []string{"1", "2", "2.5"},
			[]string{cell(t, h, recs[1], "Cue #"), cell(t, h, recs[2], "Cue #"), cell(t, h, recs[3], "Cue #")})
		assert.Equal(t, "Opening", cell(t, h, recs[1], "Name"))
		assert.Equal(t, "Look A", cell(t, h, recs[1], "Look"))
		assert.Equal(t, "2", cell(t, h, recs[1], "Fade in (s)"))
		assert.Equal(t, "Look B", cell(t, h, recs[2], "Look"))
		assert.Equal(t, "1.5", cell(t, h, recs[2], "Fade out (s)"))
		assert.Equal(t, "No", cell(t, h, recs[1], "Skip"))

		empty := csvOf(t, "CUE_LIST_DETAILS", s.ProjectID, s.EmptyCueListID)
		assert.Len(t, empty, 1, "empty cue list exports only the header row")
	})
}

// TestExportDataTable_LookDetailsWideFormat checks the wide look format:
// one row per fixture, then Ch 1 … Ch N where N is the widest fixture.
// A channel the look does not set is blank; a stored 0 is written as 0.
// The XLSX version has a second "Channel names" sheet with the same grid.
func TestExportDataTable_LookDetailsWideFormat(t *testing.T) {
	client, ctx := newClient(t)
	s := seedProject(t, ctx, client)

	chCols := []string{"Ch 1", "Ch 2", "Ch 3"}
	channels := func(t *testing.T, h, row []string) []string {
		out := make([]string, len(chCols))
		for i, c := range chCols {
			out[i] = cell(t, h, row, c)
		}
		return out
	}

	check := func(t *testing.T, records [][]string) {
		t.Helper()
		require.Len(t, records, 1+seedLookADetails)
		h := records[0]
		require.Len(t, h, len(expectedHeaders["LOOK_DETAILS"])+seedLookAChannels)
		// Rows ordered by universe / start address: Dimmer 1 (1) then RGB Par (10).
		d1, rgb := records[1], records[2]
		assert.Equal(t, "Dimmer 1", cell(t, h, d1, "Fixture"))
		assert.Equal(t, "1", cell(t, h, d1, "Start address"))
		assert.Equal(t, []string{"255", "", ""}, channels(t, h, d1),
			"channels beyond a fixture's width are blank")
		assert.Equal(t, "RGB Par", cell(t, h, rgb, "Fixture"))
		assert.Equal(t, s.Manufacturer, cell(t, h, rgb, "Manufacturer"))
		assert.Equal(t, "RGB3", cell(t, h, rgb, "Model"))
		assert.Equal(t, "10", cell(t, h, rgb, "Start address"))
		assert.Equal(t, []string{"0", "", "128"}, channels(t, h, rgb),
			"stored 0 is written as 0; a channel the look does not set is blank")
	}

	t.Run("CSV", func(t *testing.T) {
		res, err := exportTable(ctx, client, s.ProjectID, "LOOK_DETAILS", "CSV", s.LookAID)
		require.NoError(t, err)
		check(t, parseCSV(t, res))
	})

	t.Run("XLSX", func(t *testing.T) {
		res, err := exportTable(ctx, client, s.ProjectID, "LOOK_DETAILS", "XLSX", s.LookAID)
		require.NoError(t, err)
		wb := parseXLSX(t, res)
		check(t, wb.Rows["Look Details"])

		names := wb.Rows["Channel names"]
		require.Len(t, names, 1+seedLookADetails, "Channel names sheet has the same rows")
		h := names[0]
		assert.Equal(t, wb.Rows["Look Details"][0], h, "Channel names sheet has the same columns")
		assert.Equal(t, []string{"Intensity", "", ""}, channels(t, h, names[1]))
		assert.Equal(t, []string{"Red", "Green", "Blue"}, channels(t, h, names[2]))
	})
}

// TestExportDataWorkbook checks the multi-sheet XLSX export.
func TestExportDataWorkbook(t *testing.T) {
	client, ctx := newClient(t)
	s := seedProject(t, ctx, client)

	t.Run("DefaultTables", func(t *testing.T) {
		res, err := exportWorkbook(ctx, client, s.ProjectID, nil)
		require.NoError(t, err)
		assert.Equal(t, mimeXLSX, res.MimeType)
		assert.Contains(t, res.Filename, strings.ReplaceAll(s.ProjectName, " ", "-")+"-data-")
		assert.Regexp(t, datedFilename, res.Filename)

		wb := parseXLSX(t, res)
		// Default set: every project-level table that needs no scope ID,
		// excluding the global fixture library.
		want := map[string]int{
			"Fixture Types":      seedFixtureTypes,
			"Fixtures":           seedFixtures,
			"Looks":              seedLooks,
			"Effects":            seedEffects,
			"Look Board Entries": seedBoardEntriesAll,
			"Cue Lists":          seedCueLists,
		}
		require.Equal(t, []string{"Fixture Types", "Fixtures", "Looks", "Effects", "Look Board Entries", "Cue Lists"},
			wb.Sheets)
		total := 0
		for _, table := range []string{"FIXTURE_TYPES", "FIXTURES", "LOOKS", "EFFECTS", "LOOK_BOARD_ENTRIES", "CUE_LISTS"} {
			sheet := labelOf(table)
			rows := wb.Rows[sheet]
			require.NotEmpty(t, rows, "sheet %q", sheet)
			assert.Equal(t, expectedHeaders[table], rows[0], "headers of sheet %q", sheet)
			assert.Equal(t, want[sheet], len(rows)-1, "data rows in sheet %q", sheet)
			total += want[sheet]
		}
		assert.Equal(t, total, res.RowCount, "workbook rowCount is the total across sheets")
	})

	t.Run("SelectedTablesInRequestOrder", func(t *testing.T) {
		res, err := exportWorkbook(ctx, client, s.ProjectID, []string{"CUE_LISTS", "LOOKS"})
		require.NoError(t, err)
		wb := parseXLSX(t, res)
		assert.Equal(t, []string{"Cue Lists", "Looks"}, wb.Sheets)
		assert.Equal(t, seedCueLists+seedLooks, res.RowCount)
	})

	t.Run("ScopedTableRejected", func(t *testing.T) {
		_, err := exportWorkbook(ctx, client, s.ProjectID, []string{"LOOKS", "LOOK_DETAILS"})
		require.Error(t, err, "LOOK_DETAILS needs a scopeId so it cannot be part of a workbook")
		assert.Contains(t, err.Error(), "scopeId is required")
	})

	t.Run("EmptyTableListRejected", func(t *testing.T) {
		// An explicit [] is distinct from omitting tables (default set).
		_, err := exportWorkbook(ctx, client, s.ProjectID, []string{})
		require.Error(t, err, "an explicit empty table list must not fall back to the default set")
		assert.Contains(t, err.Error(), "at least one table is required")
	})

	t.Run("UnknownProjectRejected", func(t *testing.T) {
		_, err := exportWorkbook(ctx, client, "00000000-0000-0000-0000-000000000000", nil)
		require.Error(t, err)
	})
}

// TestExportDataTable_Errors checks request validation: required scope IDs,
// scope IDs from another project, and project-bound tables without a
// project.
func TestExportDataTable_Errors(t *testing.T) {
	client, ctx := newClient(t)
	s := seedProject(t, ctx, client)

	cases := []struct {
		name      string
		table     string
		projectID string
		scopeID   string
		wantErr   string
	}{
		{"LookDetailsMissingScope", "LOOK_DETAILS", s.ProjectID, "", "scopeId is required"},
		{"CueListDetailsMissingScope", "CUE_LIST_DETAILS", s.ProjectID, "", "scopeId is required"},
		{"LookFromOtherProject", "LOOK_DETAILS", s.ProjectID, s.OtherLookID, "scope item not found in project"},
		{"CueListFromOtherProject", "CUE_LIST_DETAILS", s.ProjectID, s.OtherCueListID, "scope item not found in project"},
		{"LookBoardFromOtherProject", "LOOK_BOARD_ENTRIES", s.ProjectID, s.OtherBoardID, "scope item not found in project"},
		{"NonexistentLook", "LOOK_DETAILS", s.ProjectID, "00000000-0000-0000-0000-000000000000", "scope item not found in project"},
		{"ScopeOnProjectTable", "LOOKS", s.ProjectID, s.LookAID, "scopeId is not supported"},
		{"FixturesWithoutProject", "FIXTURES", "", "", "projectId is required"},
		{"LookDetailsWithoutProject", "LOOK_DETAILS", "", s.LookAID, "projectId is required"},
		{"UnknownProject", "LOOKS", "00000000-0000-0000-0000-000000000000", "", "not found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, format := range []string{"CSV", "XLSX"} {
				res, err := exportTable(ctx, client, tc.projectID, tc.table, format, tc.scopeID)
				require.Error(t, err, "%s export should fail, got %+v", format, res)
				assert.Contains(t, err.Error(), tc.wantErr, "%s error message", format)
			}
		})
	}

	t.Run("InvalidEnum", func(t *testing.T) {
		_, err := exportTable(ctx, client, s.ProjectID, "NOT_A_TABLE", "CSV", "")
		require.Error(t, err, "unknown DataExportTable values are rejected by schema validation")
	})
}

// TestExportDataTable_CellEscaping checks that user-entered text survives
// export safely: CSV quotes commas, quotes and line breaks (RFC 4180) and
// prefixes text starting with any formula character (= + - @ TAB CR LF) with
// a single quote (CSV injection); XLSX stores the same text as a plain
// string, never as a formula.
func TestExportDataTable_CellEscaping(t *testing.T) {
	client, ctx := newClient(t)
	projectID := createProject(t, ctx, client, fmt.Sprintf("DataExport Escaping %d", time.Now().UnixNano()))
	t.Cleanup(func() { deleteProject(context.Background(), client, projectID) })

	// One look per formula-leading character the CSV export must neutralise
	// (= + - @ TAB CR LF), with the Name cell expected in the CSV. The
	// server normalises a lone CR to LF (encoding/csv with UseCRLF would
	// otherwise drop it), so "\rCR lead" is exported as "'\nCR lead".
	formulaCases := []struct{ name, csv string }{
		{"=1+1", "'=1+1"},
		{"+1+1", "'+1+1"},
		{"-1+1", "'-1+1"},
		{"@SUM(A1)", "'@SUM(A1)"},
		{"\tTab lead", "'\tTab lead"},
		{"\rCR lead", "'\nCR lead"},
		{"\nLF lead", "'\nLF lead"},
	}
	formulaNames := make([]string, 0, len(formulaCases))
	for _, c := range formulaCases {
		formulaNames = append(formulaNames, c.name)
	}
	// Text that must round-trip unchanged: commas and quotes, an embedded
	// LF (quoted, not a formula prefix), and a formula character that is
	// not in the first position.
	plainNames := []string{
		`Wash, "warm"`,
		"Line one\nLine two",
		"Total =1+1",
	}
	for _, name := range append(append([]string{}, formulaNames...), plainNames...) {
		createLook(t, ctx, client, projectID, name, []map[string]interface{}{})
	}

	t.Run("CSV", func(t *testing.T) {
		res, err := exportTable(ctx, client, projectID, "LOOKS", "CSV", "")
		require.NoError(t, err)
		byName := rowsBy(t, parseCSV(t, res), "Name")
		for _, c := range formulaCases {
			assert.Contains(t, byName, c.csv, "formula-like text %q is prefixed with a single quote", c.name)
			assert.NotContains(t, byName, c.name, "formula-like text %q must not be exported raw", c.name)
		}
		for _, name := range plainNames {
			assert.Contains(t, byName, name, "%q round-trips through CSV quoting unchanged", name)
			assert.NotContains(t, byName, "'"+name, "%q must not be quote-prefixed", name)
		}
	})

	t.Run("XLSX", func(t *testing.T) {
		res, err := exportTable(ctx, client, projectID, "LOOKS", "XLSX", "")
		require.NoError(t, err)
		f, err := excelize.OpenReader(bytes.NewReader(decodeContent(t, res)))
		require.NoError(t, err)
		defer func() { _ = f.Close() }()

		names := map[string]string{} // value -> cell
		rows, err := f.GetRows("Looks")
		require.NoError(t, err)
		for i, r := range rows[1:] {
			cellRef, _ := excelize.CoordinatesToCellName(1, i+2)
			names[r[0]] = cellRef
		}
		for _, name := range append(append([]string{}, formulaNames...), plainNames...) {
			require.Contains(t, names, name, "XLSX keeps %q as-is", name)
			formula, err := f.GetCellFormula("Looks", names[name])
			require.NoError(t, err)
			assert.Empty(t, formula, "%q must not be stored as a formula", name)
		}
	})
}
