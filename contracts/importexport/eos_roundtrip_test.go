// Eos ASCII round-trip contract test. Validates that a project created
// via native LacyLights GraphQL mutations can be exported to Eos ASCII
// and re-imported, preserving the patch.
package importexport

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEosRoundtrip_PreservesPatch builds a small project via the native
// GraphQL mutations, exports it to Eos ASCII, then imports the exported
// content into a fresh project. It asserts that the imported project
// contains at least as many fixtures as the source — i.e. the export
// writer + import parser together don't drop the patch.
//
// Validates the contract that:
//   - exportProjectToEos returns non-empty asciiContent and a filenameSuffix
//   - the writer's output is parseable by the importer (no manual editing required)
//   - patch counts survive the round trip
func TestEosRoundtrip_PreservesPatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	// Build the source project. setupExportTest is shared with the JSON
	// import/export tests in this package and creates 1 fixture, 1 look,
	// 1 cue list, and 1 cue.
	sourceProjectID := setupExportTest(t, client, ctx)
	defer deleteProject(t, client, ctx, sourceProjectID)

	// Capture source patch size for the comparison after round-trip.
	sourceFixtureCount := countProjectFixtures(t, client, ctx, sourceProjectID)
	require.GreaterOrEqual(t, sourceFixtureCount, 1, "setupExportTest should produce at least one fixture")

	// Step 1: export to Eos ASCII.
	var exp struct {
		ExportProjectToEos struct {
			ProjectID      string `json:"projectId"`
			ProjectName    string `json:"projectName"`
			ASCIIContent   string `json:"asciiContent"`
			FilenameSuffix string `json:"filenameSuffix"`
		} `json:"exportProjectToEos"`
	}
	err := client.Mutate(ctx, `
		mutation Export($p: ID!) {
			exportProjectToEos(projectId: $p) {
				projectId
				projectName
				asciiContent
				filenameSuffix
			}
		}
	`, map[string]interface{}{"p": sourceProjectID}, &exp)
	require.NoError(t, err)

	assert.Equal(t, sourceProjectID, exp.ExportProjectToEos.ProjectID)
	assert.NotEmpty(t, exp.ExportProjectToEos.ASCIIContent, "export should produce ASCII content")
	assert.NotEmpty(t, exp.ExportProjectToEos.FilenameSuffix, "export should suggest a filename suffix")

	// Sanity: ETC Eos showfiles open with an Ident header. Catches gross
	// writer regressions before we even hand the content to the importer.
	require.True(t,
		strings.Contains(exp.ExportProjectToEos.ASCIIContent, "Ident") ||
			strings.HasPrefix(strings.TrimSpace(exp.ExportProjectToEos.ASCIIContent), "!"),
		"export should look like an ETC Eos ASCII showfile")

	// Step 2: import the exported content into a fresh project.
	var imp struct {
		ImportProjectFromEos struct {
			ProjectID             string `json:"projectId"`
			FixtureInstancesCount int    `json:"fixtureInstancesCount"`
			CuesCount             int    `json:"cuesCount"`
			Warnings              []struct {
				Code     string `json:"code"`
				Severity string `json:"severity"`
			} `json:"warnings"`
		} `json:"importProjectFromEos"`
	}
	err = client.Mutate(ctx, `
		mutation Import($c: String!, $opts: EosImportOptionsInput) {
			importProjectFromEos(asciiContent: $c, options: $opts) {
				projectId
				fixtureInstancesCount
				cuesCount
				warnings { code severity }
			}
		}
	`, map[string]interface{}{
		"c": exp.ExportProjectToEos.ASCIIContent,
		"opts": map[string]interface{}{
			"newProjectName": "Eos Roundtrip Imported",
		},
	}, &imp)
	require.NoError(t, err)
	defer deleteProject(t, client, ctx, imp.ImportProjectFromEos.ProjectID)

	// Step 3: compare. Patch must round-trip exactly; the writer's job is
	// to emit every fixture in a form the parser can re-create.
	assert.NotEmpty(t, imp.ImportProjectFromEos.ProjectID, "import should create a new project")
	assert.Equal(t, sourceFixtureCount, imp.ImportProjectFromEos.FixtureInstancesCount,
		"round-trip should preserve fixture count: source=%d, imported=%d",
		sourceFixtureCount, imp.ImportProjectFromEos.FixtureInstancesCount)
}

// countProjectFixtures returns the number of fixture instances on the
// given project. Used to compare patch sizes across the round-trip.
func countProjectFixtures(t *testing.T, client *graphql.Client, ctx context.Context, projectID string) int {
	t.Helper()
	var resp struct {
		Project struct {
			Fixtures []struct {
				ID string `json:"id"`
			} `json:"fixtures"`
		} `json:"project"`
	}
	err := client.Query(ctx, `
		query CountFixtures($id: ID!) {
			project(id: $id) {
				fixtures { id }
			}
		}
	`, map[string]interface{}{"id": projectID}, &resp)
	require.NoError(t, err)
	return len(resp.Project.Fixtures)
}
