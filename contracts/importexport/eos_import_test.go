// Eos ASCII import contract tests. Validates that posting a real-world
// ETC Eos showfile through the importProjectFromEos GraphQL mutation
// populates project state and surfaces structured warnings for
// EOS-only directives that LacyLights deliberately drops on import.
package importexport

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eosFixturePath returns the absolute path to the OTBPA golden fixture in
// the top-level lacylights aggregator directory.
//
// Resolution order:
//  1. EOS_ASCII_FIXTURE env var (allows CI to point at an alternate location)
//  2. ../../../docs/OTBPA DRESS TUES.asc relative to this test file
func eosFixturePath(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("EOS_ASCII_FIXTURE"); p != "" {
		return p
	}
	cwd, err := os.Getwd()
	require.NoError(t, err)
	// cwd is contracts/importexport; the aggregator lives three levels up.
	return filepath.Join(cwd, "..", "..", "..", "docs", "OTBPA DRESS TUES.asc")
}

// loadEosFixture reads the golden fixture or skips if it isn't checked out.
// The aggregator directory containing the fixture is not part of this repo,
// so CI environments that clone only lacylights-test will skip these tests.
func loadEosFixture(t *testing.T) string {
	t.Helper()
	path := eosFixturePath(t)
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("skipping: Eos golden fixture not available at %s: %v", path, err)
	}
	return string(bytes)
}

// deleteProject is a best-effort cleanup helper for tests that create projects.
func deleteProject(t *testing.T, client *graphql.Client, ctx context.Context, projectID string) {
	t.Helper()
	if projectID == "" {
		return
	}
	_ = client.Mutate(ctx, `mutation DeleteProject($id: ID!) { deleteProject(id: $id) }`,
		map[string]interface{}{"id": projectID}, nil)
}

// TestEosImport_PopulatesProject posts the OTBPA showfile through the
// importProjectFromEos mutation and asserts that the resulting project
// contains a meaningful number of fixtures.
//
// Validates the contract that:
//   - importProjectFromEos accepts asciiContent and an optional EosImportOptionsInput
//   - the result returns projectId and per-entity counts
//   - warnings are returned as structured records (code/severity/message)
func TestEosImport_PopulatesProject(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	asciiContent := loadEosFixture(t)
	client := graphql.NewClient("")

	var resp struct {
		ImportProjectFromEos struct {
			ProjectID               string `json:"projectId"`
			FixtureDefinitionsCount int    `json:"fixtureDefinitionsCount"`
			FixtureInstancesCount   int    `json:"fixtureInstancesCount"`
			LooksCount              int    `json:"looksCount"`
			CueListsCount           int    `json:"cueListsCount"`
			CuesCount               int    `json:"cuesCount"`
			GroupsCount             int    `json:"groupsCount"`
			Warnings                []struct {
				Code     string `json:"code"`
				Severity string `json:"severity"`
				Message  string `json:"message"`
			} `json:"warnings"`
			SynthesizedDefinitionIds []string `json:"synthesizedDefinitionIds"`
		} `json:"importProjectFromEos"`
	}

	err := client.Mutate(ctx, `
		mutation Import($c: String!, $opts: EosImportOptionsInput) {
			importProjectFromEos(asciiContent: $c, options: $opts) {
				projectId
				fixtureDefinitionsCount
				fixtureInstancesCount
				looksCount
				cueListsCount
				cuesCount
				groupsCount
				warnings { code severity message }
				synthesizedDefinitionIds
			}
		}
	`, map[string]interface{}{
		"c": asciiContent,
		"opts": map[string]interface{}{
			"newProjectName": "Eos Import Contract Test",
		},
	}, &resp)

	require.NoError(t, err)
	defer deleteProject(t, client, ctx, resp.ImportProjectFromEos.ProjectID)

	// OTBPA's patch is dozens of fixtures across multiple universes.
	// Use a generous floor that catches "import silently dropped most
	// of the patch" without being brittle to fixture-library matching
	// shifts (matched fixtures yield instances; unmatched fixtures still
	// produce instances via synthesized definitions).
	assert.NotEmpty(t, resp.ImportProjectFromEos.ProjectID, "expected a created project")
	assert.GreaterOrEqual(t, resp.ImportProjectFromEos.FixtureInstancesCount, 20,
		"expected the OTBPA showfile to import at least 20 fixtures")
	assert.GreaterOrEqual(t, resp.ImportProjectFromEos.CuesCount, 1,
		"expected at least one cue to be imported")

	// EOS-only constructs that LacyLights deliberately doesn't model
	// (effects, partitions, magic sheets, action triggers, curves, etc.)
	// should each emit a structured warning rather than silently disappear.
	// We don't pin a specific code because the OTBPA fixture's mix of
	// directives can shift between exports; we just assert that the
	// import surfaces at least one such warning.
	assert.NotEmpty(t, resp.ImportProjectFromEos.Warnings,
		"expected structured warnings for EOS-only directives skipped on import")
}

// TestEosImport_RejectsConflictingOptions validates the backend's input
// validation: the EosImportOptionsInput documents that newProjectName and
// targetProjectId are mutually exclusive.
func TestEosImport_RejectsConflictingOptions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	asciiContent := loadEosFixture(t)
	client := graphql.NewClient("")

	// Create a target project so targetProjectId resolves to a real ID;
	// the validation should still kick in regardless of project existence.
	var projResp struct {
		CreateProject struct {
			ID string `json:"id"`
		} `json:"createProject"`
	}
	require.NoError(t, client.Mutate(ctx, `
		mutation Create($input: CreateProjectInput!) {
			createProject(input: $input) { id }
		}
	`, map[string]interface{}{
		"input": map[string]interface{}{
			"name":        "Eos Conflict Options Test",
			"description": "target for conflicting-options test",
		},
	}, &projResp))
	defer deleteProject(t, client, ctx, projResp.CreateProject.ID)

	var resp struct {
		ImportProjectFromEos struct {
			ProjectID string `json:"projectId"`
		} `json:"importProjectFromEos"`
	}

	err := client.Mutate(ctx, `
		mutation Import($c: String!, $opts: EosImportOptionsInput) {
			importProjectFromEos(asciiContent: $c, options: $opts) {
				projectId
			}
		}
	`, map[string]interface{}{
		"c": asciiContent,
		"opts": map[string]interface{}{
			"newProjectName":  "Should Not Be Created",
			"targetProjectId": projResp.CreateProject.ID,
		},
	}, &resp)

	require.Error(t, err, "expected backend to reject mutually-exclusive options")
	// Best-effort cleanup if the backend created something despite the error.
	deleteProject(t, client, ctx, resp.ImportProjectFromEos.ProjectID)

	msg := strings.ToLower(err.Error())
	assert.True(t,
		strings.Contains(msg, "newprojectname") ||
			strings.Contains(msg, "targetprojectid") ||
			strings.Contains(msg, "mutually") ||
			strings.Contains(msg, "exclusive"),
		"expected error to mention the conflicting fields, got: %s", err.Error())
}
