// Package dataexport contains contract tests for the CSV / XLSX data-list
// export API (Query.dataExportTables, Query.exportDataTable and
// Query.exportDataWorkbook) served by lacylights-go and consumed by
// lacylights-fe (Import / Export page) and lacylights-mcp.
package dataexport

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bbernstein/lacylights-test/pkg/graphql"
	"github.com/stretchr/testify/require"
)

// seededProject describes the data created by seedProject. Expected row
// counts in the tests are derived from this layout:
//
//	Fixture definitions (global, unique manufacturer per run):
//	  <Manufacturer> Dimmer1  DIMMER   1 channel  (Intensity)
//	  <Manufacturer> RGB3     LED_PAR  3 channels (Red, Green, Blue)
//	Fixtures (3): Dimmer 1 (U1/1, tag "front"), Dimmer 2 (U1/2), RGB Par (U1/10)
//	Fixture group "Front Wash": Dimmer 1, Dimmer 2
//	Looks (2):
//	  Look A: Dimmer 1 ch1=255; RGB Par ch1=0, ch3=128 (ch2 unset)
//	  Look B: Dimmer 2 ch1=100
//	Effects (2): Pulse (WAVEFORM, 1 fixture), Hold (STATIC, 0 fixtures)
//	Look boards (2): Main Board (2 buttons), Side Board (1 button, fade 1.5s)
//	Cue lists (2): Main Show (3 cues), Empty List (0 cues)
//
// A second project ("other") owns one look, one cue list and one look
// board; it is used to check that scope IDs from another project are
// rejected.
type seededProject struct {
	ProjectID    string
	ProjectName  string
	Manufacturer string

	DimmerDefID string
	RGBDefID    string

	Dimmer1ID string
	Dimmer2ID string
	RGBParID  string

	LookAID string
	LookBID string

	MainBoardID string
	SideBoardID string

	MainCueListID  string
	EmptyCueListID string

	OtherProjectID string
	OtherLookID    string
	OtherCueListID string
	OtherBoardID   string
}

// Expected row counts for the seeded project.
const (
	seedFixtureTypes     = 2
	seedFixtures         = 3
	seedLooks            = 2
	seedLookADetails     = 2 // Dimmer 1 + RGB Par
	seedLookAChannels    = 3 // widest fixture in Look A (RGB Par)
	seedEffects          = 2
	seedBoardEntriesAll  = 3
	seedBoardEntriesMain = 2
	seedCueLists         = 2
	seedMainCues         = 3
)

func mutateID(t *testing.T, ctx context.Context, client *graphql.Client, field, mutation string, vars map[string]interface{}) string {
	t.Helper()
	var resp map[string]struct {
		ID string `json:"id"`
	}
	require.NoError(t, client.Mutate(ctx, mutation, vars, &resp), "%s failed", field)
	id := resp[field].ID
	require.NotEmpty(t, id, "%s returned no id", field)
	return id
}

func createProject(t *testing.T, ctx context.Context, client *graphql.Client, name string) string {
	t.Helper()
	return mutateID(t, ctx, client, "createProject", `
		mutation($input: CreateProjectInput!) { createProject(input: $input) { id } }
	`, map[string]interface{}{"input": map[string]interface{}{"name": name}})
}

// deleteProject removes a test project for good: deleteProject only
// soft-deletes, and the soft-deleted fixtures would keep referencing (and
// so block deletion of) the global fixture definitions, so the project is
// then permanently deleted. Best effort; errors are ignored.
func deleteProject(ctx context.Context, client *graphql.Client, id string) {
	if id == "" {
		return
	}
	vars := map[string]interface{}{"id": id}
	_ = client.Mutate(ctx, `mutation($id: ID!) { deleteProject(id: $id) }`, vars, nil)
	_ = client.Mutate(ctx, `mutation($id: ID!) { permanentlyDeleteProject(id: $id) }`, vars, nil)
}

func createDefinition(t *testing.T, ctx context.Context, client *graphql.Client, manufacturer, model, fixtureType string, channels []map[string]interface{}) string {
	t.Helper()
	return mutateID(t, ctx, client, "createFixtureDefinition", `
		mutation($input: CreateFixtureDefinitionInput!) { createFixtureDefinition(input: $input) { id } }
	`, map[string]interface{}{"input": map[string]interface{}{
		"manufacturer": manufacturer,
		"model":        model,
		"type":         fixtureType,
		"channels":     channels,
	}})
}

func channelDef(name, channelType string, offset int) map[string]interface{} {
	return map[string]interface{}{
		"name": name, "type": channelType, "offset": offset,
		"minValue": 0, "maxValue": 255, "defaultValue": 0,
	}
}

func createFixture(t *testing.T, ctx context.Context, client *graphql.Client, projectID, defID, name string, startChannel int, tags []string) string {
	t.Helper()
	input := map[string]interface{}{
		"projectId":    projectID,
		"definitionId": defID,
		"name":         name,
		"universe":     1,
		"startChannel": startChannel,
	}
	if tags != nil {
		input["tags"] = tags
	}
	return mutateID(t, ctx, client, "createFixtureInstance", `
		mutation($input: CreateFixtureInstanceInput!) { createFixtureInstance(input: $input) { id } }
	`, map[string]interface{}{"input": input})
}

// fixtureValue builds a sparse FixtureValueInput from offset/value pairs.
func fixtureValue(fixtureID string, offsetValues ...int) map[string]interface{} {
	chans := []map[string]interface{}{}
	for i := 0; i+1 < len(offsetValues); i += 2 {
		chans = append(chans, map[string]interface{}{"offset": offsetValues[i], "value": offsetValues[i+1]})
	}
	return map[string]interface{}{"fixtureId": fixtureID, "channels": chans}
}

func createLook(t *testing.T, ctx context.Context, client *graphql.Client, projectID, name string, values []map[string]interface{}) string {
	t.Helper()
	return mutateID(t, ctx, client, "createLook", `
		mutation($input: CreateLookInput!) { createLook(input: $input) { id } }
	`, map[string]interface{}{"input": map[string]interface{}{
		"projectId": projectID, "name": name, "fixtureValues": values,
	}})
}

func createBoard(t *testing.T, ctx context.Context, client *graphql.Client, projectID, name string, fade float64) string {
	t.Helper()
	return mutateID(t, ctx, client, "createLookBoard", `
		mutation($input: CreateLookBoardInput!) { createLookBoard(input: $input) { id } }
	`, map[string]interface{}{"input": map[string]interface{}{
		"projectId": projectID, "name": name, "defaultFadeTime": fade,
	}})
}

func addButton(t *testing.T, ctx context.Context, client *graphql.Client, boardID, lookID string, x, y int, label string) {
	t.Helper()
	mutateID(t, ctx, client, "addLookToBoard", `
		mutation($input: CreateLookBoardButtonInput!) { addLookToBoard(input: $input) { id } }
	`, map[string]interface{}{"input": map[string]interface{}{
		"lookBoardId": boardID, "lookId": lookID, "layoutX": x, "layoutY": y, "label": label,
	}})
}

func createCueList(t *testing.T, ctx context.Context, client *graphql.Client, projectID, name string) string {
	t.Helper()
	return mutateID(t, ctx, client, "createCueList", `
		mutation($input: CreateCueListInput!) { createCueList(input: $input) { id } }
	`, map[string]interface{}{"input": map[string]interface{}{"projectId": projectID, "name": name}})
}

func createCue(t *testing.T, ctx context.Context, client *graphql.Client, cueListID, lookID, name string, number, fadeIn, fadeOut float64) {
	t.Helper()
	mutateID(t, ctx, client, "createCue", `
		mutation($input: CreateCueInput!) { createCue(input: $input) { id } }
	`, map[string]interface{}{"input": map[string]interface{}{
		"cueListId": cueListID, "lookId": lookID, "name": name,
		"cueNumber": number, "fadeInTime": fadeIn, "fadeOutTime": fadeOut,
	}})
}

// seedProject creates the data described on seededProject and registers
// cleanup of both projects and the fixture definitions.
func seedProject(t *testing.T, ctx context.Context, client *graphql.Client) *seededProject {
	t.Helper()
	suffix := time.Now().UnixNano()
	s := &seededProject{
		ProjectName:  fmt.Sprintf("DataExport Contract %d", suffix),
		Manufacturer: fmt.Sprintf("DataExportTest%d", suffix),
	}

	s.ProjectID = createProject(t, ctx, client, s.ProjectName)
	s.OtherProjectID = createProject(t, ctx, client, fmt.Sprintf("DataExport Other %d", suffix))
	t.Cleanup(func() {
		cctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		deleteProject(cctx, client, s.ProjectID)
		deleteProject(cctx, client, s.OtherProjectID)
		// The projects are gone for good, so nothing references the
		// per-run definitions any more; failing to delete them would leak
		// two global FIXTURE_LIBRARY rows per run.
		for _, id := range []string{s.DimmerDefID, s.RGBDefID} {
			if id == "" {
				continue
			}
			if err := client.Mutate(cctx, `mutation($id: ID!) { deleteFixtureDefinition(id: $id) }`,
				map[string]interface{}{"id": id}, nil); err != nil {
				t.Errorf("cleanup: deleting fixture definition %s: %v", id, err)
			}
		}
	})

	// Fixture definitions.
	s.DimmerDefID = createDefinition(t, ctx, client, s.Manufacturer, "Dimmer1", "DIMMER",
		[]map[string]interface{}{channelDef("Intensity", "INTENSITY", 0)})
	s.RGBDefID = createDefinition(t, ctx, client, s.Manufacturer, "RGB3", "LED_PAR",
		[]map[string]interface{}{
			channelDef("Red", "RED", 0),
			channelDef("Green", "GREEN", 1),
			channelDef("Blue", "BLUE", 2),
		})

	// Fixtures and a fixture group.
	s.Dimmer1ID = createFixture(t, ctx, client, s.ProjectID, s.DimmerDefID, "Dimmer 1", 1, []string{"front"})
	s.Dimmer2ID = createFixture(t, ctx, client, s.ProjectID, s.DimmerDefID, "Dimmer 2", 2, nil)
	s.RGBParID = createFixture(t, ctx, client, s.ProjectID, s.RGBDefID, "RGB Par", 10, nil)
	mutateID(t, ctx, client, "createFixtureGroup", `
		mutation($input: CreateFixtureGroupInput!) { createFixtureGroup(input: $input) { id } }
	`, map[string]interface{}{"input": map[string]interface{}{
		"projectId": s.ProjectID, "name": "Front Wash",
		"fixtureIds": []string{s.Dimmer1ID, s.Dimmer2ID},
	}})

	// Looks. Look A has an explicit 0 (RGB Par ch1) and an unset channel
	// (RGB Par ch2) so the wide LOOK_DETAILS format can be checked.
	s.LookAID = createLook(t, ctx, client, s.ProjectID, "Look A", []map[string]interface{}{
		fixtureValue(s.Dimmer1ID, 0, 255),
		fixtureValue(s.RGBParID, 0, 0, 2, 128),
	})
	s.LookBID = createLook(t, ctx, client, s.ProjectID, "Look B", []map[string]interface{}{
		fixtureValue(s.Dimmer2ID, 0, 100),
	})

	// Effects.
	pulseID := mutateID(t, ctx, client, "createEffect", `
		mutation($input: CreateEffectInput!) { createEffect(input: $input) { id } }
	`, map[string]interface{}{"input": map[string]interface{}{
		"projectId": s.ProjectID, "name": "Pulse", "effectType": "WAVEFORM",
		"waveform": "SINE", "frequency": 2.0, "amplitude": 80.0,
	}})
	mutateID(t, ctx, client, "addFixtureToEffect", `
		mutation($input: AddFixtureToEffectInput!) { addFixtureToEffect(input: $input) { id } }
	`, map[string]interface{}{"input": map[string]interface{}{"effectId": pulseID, "fixtureId": s.Dimmer1ID}})
	mutateID(t, ctx, client, "createEffect", `
		mutation($input: CreateEffectInput!) { createEffect(input: $input) { id } }
	`, map[string]interface{}{"input": map[string]interface{}{
		"projectId": s.ProjectID, "name": "Hold", "effectType": "STATIC",
	}})

	// Look boards.
	s.MainBoardID = createBoard(t, ctx, client, s.ProjectID, "Main Board", 3.0)
	addButton(t, ctx, client, s.MainBoardID, s.LookAID, 0, 0, "A")
	addButton(t, ctx, client, s.MainBoardID, s.LookBID, 250, 0, "B")
	s.SideBoardID = createBoard(t, ctx, client, s.ProjectID, "Side Board", 1.5)
	addButton(t, ctx, client, s.SideBoardID, s.LookAID, 0, 0, "Side A")

	// Cue lists.
	s.MainCueListID = createCueList(t, ctx, client, s.ProjectID, "Main Show")
	createCue(t, ctx, client, s.MainCueListID, s.LookAID, "Opening", 1, 2, 1)
	createCue(t, ctx, client, s.MainCueListID, s.LookBID, "Scene Two", 2, 3, 1.5)
	createCue(t, ctx, client, s.MainCueListID, s.LookAID, "Point Five", 2.5, 0, 0)
	s.EmptyCueListID = createCueList(t, ctx, client, s.ProjectID, "Empty List")

	// Items in another project, for cross-project scope checks.
	s.OtherLookID = createLook(t, ctx, client, s.OtherProjectID, "Other Look", []map[string]interface{}{})
	s.OtherCueListID = createCueList(t, ctx, client, s.OtherProjectID, "Other Cue List")
	s.OtherBoardID = createBoard(t, ctx, client, s.OtherProjectID, "Other Board", 3.0)

	return s
}
