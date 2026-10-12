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

// TestProjectExportQueriesReplaceDeprecatedMutations validates the
// migration contract for project exports (lacylights-fe#175):
//
//   - Query.projectExport / projectExportQLC / projectExportEos exist.
//   - Mutation.exportProject / exportProjectToQLC / exportProjectToEos are
//     marked @deprecated but still work, because Mac and RPi builds ship
//     older frontends that call them. They must return the same data as
//     the Queries until they are removed in a later release.
//
// When the deprecated Mutations are removed from lacylights-go, delete the
// "DeprecatedMutationsStillWork" subtest and the deprecation checks.
func TestProjectExportQueriesReplaceDeprecatedMutations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := graphql.NewClient("")

	t.Run("SchemaDeprecation", func(t *testing.T) {
		var resp struct {
			Query struct {
				Fields []struct {
					Name         string `json:"name"`
					IsDeprecated bool   `json:"isDeprecated"`
				} `json:"fields"`
			} `json:"query"`
			Mutation struct {
				Fields []struct {
					Name              string  `json:"name"`
					IsDeprecated      bool    `json:"isDeprecated"`
					DeprecationReason *string `json:"deprecationReason"`
				} `json:"fields"`
			} `json:"mutation"`
		}
		err := client.Query(ctx, `
			query {
				query: __type(name: "Query") { fields(includeDeprecated: true) { name isDeprecated } }
				mutation: __type(name: "Mutation") {
					fields(includeDeprecated: true) { name isDeprecated deprecationReason }
				}
			}
		`, nil, &resp)
		require.NoError(t, err)

		queries := map[string]bool{}
		for _, f := range resp.Query.Fields {
			queries[f.Name] = f.IsDeprecated
		}
		for _, name := range []string{"projectExport", "projectExportQLC", "projectExportEos"} {
			deprecated, ok := queries[name]
			assert.True(t, ok, "Query.%s must exist", name)
			assert.False(t, deprecated, "Query.%s must not be deprecated", name)
		}

		mutations := map[string]bool{}
		for _, f := range resp.Mutation.Fields {
			if f.IsDeprecated {
				mutations[f.Name] = true
				require.NotNil(t, f.DeprecationReason)
				if strings.HasPrefix(f.Name, "exportProject") {
					assert.Contains(t, *f.DeprecationReason, "projectExport",
						"deprecation reason for %s should point at the replacement Query", f.Name)
				}
			}
		}
		for _, name := range []string{"exportProject", "exportProjectToQLC", "exportProjectToEos"} {
			assert.True(t, mutations[name], "Mutation.%s must still exist and be marked deprecated", name)
		}
	})

	t.Run("DeprecatedMutationsStillWork", func(t *testing.T) {
		projectID := setupExportTest(t, client, ctx)
		defer deleteProject(t, client, ctx, projectID)

		// JSON: same project and stats from the Mutation and the Query.
		type jsonExport struct {
			ProjectID   string `json:"projectId"`
			ProjectName string `json:"projectName"`
			JSONContent string `json:"jsonContent"`
			Stats       struct {
				FixtureInstancesCount int `json:"fixtureInstancesCount"`
				LooksCount            int `json:"looksCount"`
				CueListsCount         int `json:"cueListsCount"`
				CuesCount             int `json:"cuesCount"`
			} `json:"stats"`
		}
		const jsonFields = `projectId projectName jsonContent
			stats { fixtureInstancesCount looksCount cueListsCount cuesCount }`
		vars := map[string]interface{}{"projectId": projectID}

		var mutResp struct {
			ExportProject jsonExport `json:"exportProject"`
		}
		require.NoError(t, client.Mutate(ctx,
			`mutation($projectId: ID!) { exportProject(projectId: $projectId) { `+jsonFields+` } }`,
			vars, &mutResp))
		var queryResp struct {
			ProjectExport jsonExport `json:"projectExport"`
		}
		require.NoError(t, client.Query(ctx,
			`query($projectId: ID!) { projectExport(projectId: $projectId) { `+jsonFields+` } }`,
			vars, &queryResp))

		m, q := mutResp.ExportProject, queryResp.ProjectExport
		assert.Equal(t, projectID, m.ProjectID)
		assert.Equal(t, q.ProjectID, m.ProjectID)
		assert.Equal(t, q.ProjectName, m.ProjectName)
		assert.Equal(t, q.Stats, m.Stats, "Mutation and Query must report the same stats")
		assert.NotEmpty(t, m.JSONContent)
		assert.GreaterOrEqual(t, m.Stats.FixtureInstancesCount, 1)

		// Eos: both return an ASCII showfile for the same project.
		type eosExport struct {
			ProjectID    string `json:"projectId"`
			ASCIIContent string `json:"asciiContent"`
		}
		var eosMut struct {
			ExportProjectToEos eosExport `json:"exportProjectToEos"`
		}
		require.NoError(t, client.Mutate(ctx,
			`mutation($projectId: ID!) { exportProjectToEos(projectId: $projectId) { projectId asciiContent } }`,
			vars, &eosMut))
		var eosQuery struct {
			ProjectExportEos eosExport `json:"projectExportEos"`
		}
		require.NoError(t, client.Query(ctx,
			`query($projectId: ID!) { projectExportEos(projectId: $projectId) { projectId asciiContent } }`,
			vars, &eosQuery))
		assert.Equal(t, projectID, eosMut.ExportProjectToEos.ProjectID)
		assert.NotEmpty(t, eosMut.ExportProjectToEos.ASCIIContent)
		assert.Equal(t, eosQuery.ProjectExportEos.ProjectID, eosMut.ExportProjectToEos.ProjectID)

		// QLC+: same counts from both, unless QLC+ export is unavailable on
		// this platform (then both must fail the same way).
		type qlcExport struct {
			FixtureCount int    `json:"fixtureCount"`
			LookCount    int    `json:"lookCount"`
			CueListCount int    `json:"cueListCount"`
			XMLContent   string `json:"xmlContent"`
		}
		const qlcFields = `fixtureCount lookCount cueListCount xmlContent`
		var qlcMut struct {
			ExportProjectToQLC qlcExport `json:"exportProjectToQLC"`
		}
		mutErr := client.Mutate(ctx,
			`mutation($projectId: ID!) { exportProjectToQLC(projectId: $projectId) { `+qlcFields+` } }`,
			vars, &qlcMut)
		var qlcQuery struct {
			ProjectExportQLC qlcExport `json:"projectExportQLC"`
		}
		queryErr := client.Query(ctx,
			`query($projectId: ID!) { projectExportQLC(projectId: $projectId) { `+qlcFields+` } }`,
			vars, &qlcQuery)
		if mutErr != nil && strings.Contains(mutErr.Error(), "not available") {
			require.Error(t, queryErr, "Query must fail like the Mutation when QLC+ is unavailable")
			return
		}
		require.NoError(t, mutErr)
		require.NoError(t, queryErr)
		assert.Equal(t, qlcQuery.ProjectExportQLC.FixtureCount, qlcMut.ExportProjectToQLC.FixtureCount)
		assert.Equal(t, qlcQuery.ProjectExportQLC.LookCount, qlcMut.ExportProjectToQLC.LookCount)
		assert.Equal(t, qlcQuery.ProjectExportQLC.CueListCount, qlcMut.ExportProjectToQLC.CueListCount)
		assert.NotEmpty(t, qlcMut.ExportProjectToQLC.XMLContent)
	})
}
