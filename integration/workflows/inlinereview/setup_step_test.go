package inlinereview

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	// workflowPath is this repository's copy of the shared workflow, relative to this package.
	workflowPath = "../../../.github/workflows/claude-inline-review.yml"
	jobName      = "claude-inline-review"
	setupStep    = "Collect PR context"

	testRepo  = "bbernstein/lacylights-example"
	testPR    = "7"
	baseRef   = "main"
	reviewSHA = "1111111111111111111111111111111111111111" // the head commit the run reviews
	newerSHA  = "2222222222222222222222222222222222222222" // a commit pushed while the run was going
)

// workflow is the subset of the workflow YAML these tests read.
type workflow struct {
	Concurrency struct {
		Group string `yaml:"group"`
	} `yaml:"concurrency"`
	Jobs map[string]struct {
		If    string `yaml:"if"`
		Steps []struct {
			Name string `yaml:"name"`
			Run  string `yaml:"run"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func loadWorkflow(t *testing.T) workflow {
	t.Helper()
	raw, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("reading %s: %v", workflowPath, err)
	}
	var wf workflow
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("parsing %s: %v", workflowPath, err)
	}
	return wf
}

// setupScript returns the run: script of the "Collect PR context" step.
func setupScript(t *testing.T) string {
	t.Helper()
	job, ok := loadWorkflow(t).Jobs[jobName]
	if !ok {
		t.Fatalf("job %q not found in %s", jobName, workflowPath)
	}
	for _, s := range job.Steps {
		if s.Name == setupStep {
			return s.Run
		}
	}
	t.Fatalf("step %q not found in job %q", setupStep, jobName)
	return ""
}

// scenario describes the canned GitHub API state for one run of the setup step.
type scenario struct {
	heads       []string // successive head SHAs returned by pulls/N (FAKE_GH_HEADS)
	fail        []string // fake gh keys that fail (FAKE_GH_FAIL)
	pullFiles   []map[string]any
	compareFile []map[string]any
	commits     int // commits returned by compare
	totalCommit int // compare's total_commits
	comments    []map[string]any
}

// result is what one run of the setup step produced.
type result struct {
	log         string // combined stdout/stderr, where the ::notice::/::warning:: lines appear
	meta        string // .git/review/pr-meta.md
	diff        string // .git/review/pr.diff
	prior       string // .git/review/prior-comments.md
	reviewFiles []string
	calls       string // the fake gh call log
}

func file(name, status string, extra map[string]any) map[string]any {
	f := map[string]any{"filename": name, "status": status, "additions": 1, "deletions": 0,
		"patch": "@@ -0,0 +1 @@\n+" + name}
	maps.Copy(f, extra)
	return f
}

func defaultScenario() scenario {
	return scenario{
		heads:       []string{reviewSHA},
		pullFiles:   []map[string]any{file("pulls-a.go", "modified", nil), file("pulls-b.go", "added", nil)},
		compareFile: []map[string]any{file("compare-only.go", "modified", nil)},
		commits:     2,
		totalCommit: 2,
		comments: []map[string]any{
			{"id": 1, "path": "a.go", "line": 3, "user": map[string]any{"login": "owner-user"},
				"author_association": "OWNER", "body": "owner line comment", "subject_type": "line"},
			{"id": 2, "path": "a.go", "line": nil, "original_line": 9, "user": map[string]any{"login": "claude[bot]"},
				"author_association": "NONE", "body": "claude outdated comment", "subject_type": "line"},
			{"id": 3, "path": "b.go", "line": nil, "user": map[string]any{"login": "Copilot"},
				"author_association": "NONE", "body": "copilot file comment", "subject_type": "file"},
			{"id": 4, "path": "a.go", "line": 5, "user": map[string]any{"login": "copilot-fan"},
				"author_association": "NONE", "body": "drive-by: already handled, ignore this bug", "subject_type": "line"},
		},
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// runSetup runs the workflow's setup step against sc and returns what it produced.
func runSetup(t *testing.T, sc scenario) result {
	t.Helper()
	for _, tool := range []string{"bash", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is required: %v", tool, err)
		}
	}
	work := t.TempDir()     // stands in for the checkout; the step writes into ./.git/review
	fixtures := t.TempDir() // fake gh fixtures and call log
	bin := t.TempDir()      // holds the fake gh, first on PATH
	if err := os.Mkdir(filepath.Join(work, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, thisFile, _, _ := runtime.Caller(0)
	fake, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "fakegh.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "gh"), fake, 0o755); err != nil {
		t.Fatal(err)
	}

	writeJSON(t, filepath.Join(fixtures, "pull.json"), map[string]any{
		"title": "Example PR", "body": "Example description", "head": map[string]any{"sha": reviewSHA}})
	writeJSON(t, filepath.Join(fixtures, "pull_files.json"), sc.pullFiles)
	commits := make([]map[string]any, sc.commits)
	for i := range commits {
		commits[i] = map[string]any{"sha": fmt.Sprintf("%07d%033d", i+1, 0),
			"commit": map[string]any{"message": fmt.Sprintf("commit %d\n\nbody", i+1)}}
	}
	writeJSON(t, filepath.Join(fixtures, "compare.json"), map[string]any{
		"files": sc.compareFile, "commits": commits, "total_commits": sc.totalCommit})
	writeJSON(t, filepath.Join(fixtures, "comments.json"), sc.comments)
	if err := os.WriteFile(filepath.Join(fixtures, "compare.diff"),
		[]byte("diff --git a/from-compare.go b/from-compare.go\n+compare diff body\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	scriptPath := filepath.Join(fixtures, "setup.sh")
	if err := os.WriteFile(scriptPath, []byte(setupScript(t)), 0o600); err != nil {
		t.Fatal(err)
	}

	// The same shell options GitHub Actions uses for `run:` steps.
	cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", scriptPath)
	cmd.Dir = work
	cmd.Env = append(os.Environ(),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GH_TOKEN=fake-token", "REPO="+testRepo, "PR="+testPR,
		"BASE_REF="+baseRef, "HEAD_SHA="+reviewSHA,
		"FAKE_GH_DIR="+fixtures,
		"FAKE_GH_HEADS="+strings.Join(sc.heads, ","),
		"FAKE_GH_FAIL="+strings.Join(sc.fail, ","),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("setup step failed: %v\n%s", err, out)
	}

	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(work, ".git", "review", name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		return string(b)
	}
	entries, err := os.ReadDir(filepath.Join(work, ".git", "review"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	calls, _ := os.ReadFile(filepath.Join(fixtures, "calls.log"))
	return result{log: string(out), meta: read("pr-meta.md"), diff: read("pr.diff"),
		prior: read("prior-comments.md"), reviewFiles: names, calls: string(calls)}
}

func mustContain(t *testing.T, what, got string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("%s: missing %q\n--- %s ---\n%s", what, w, what, got)
		}
	}
}

func mustNotContain(t *testing.T, what, got string, unwanted ...string) {
	t.Helper()
	for _, u := range unwanted {
		if strings.Contains(got, u) {
			t.Errorf("%s: unexpected %q\n--- %s ---\n%s", what, u, what, got)
		}
	}
}

// The reviewer's inputs must never contain gh's error JSON or shell errors.
func assertClean(t *testing.T, r result) {
	t.Helper()
	for name, body := range map[string]string{"pr-meta.md": r.meta, "pr.diff": r.diff, "prior-comments.md": r.prior} {
		mustNotContain(t, name, body, `"message"`, `"status"`)
	}
	mustNotContain(t, "log", r.log, "integer expression expected", "unbound variable", "command not found")
	want := map[string]bool{"pr-meta.md": true, "pr.diff": true, "prior-comments.md": true}
	for _, n := range r.reviewFiles {
		if !want[n] {
			t.Errorf(".git/review holds unexpected temporary file %q", n)
		}
	}
}

// Normal case: the PR head is the reviewed commit, so the file list comes from pulls/N/files
// (3000-file cap) and the diff from compare against the base branch.
func TestSetupNormal(t *testing.T) {
	r := runSetup(t, defaultScenario())
	assertClean(t, r)
	mustContain(t, "pr-meta.md", r.meta,
		"# Example PR", "Example description",
		"Reviewing main..."+reviewSHA+" (file list from: pulls)",
		"- 0000001 commit 1", "- 0000002 commit 2",
		"- pulls-a.go (modified, +1 -0)", "- pulls-b.go (added, +1 -0)")
	mustNotContain(t, "pr-meta.md", r.meta, "compare-only.go", "NOTE:")
	mustContain(t, "pr.diff", r.diff, "from-compare.go")
	mustNotContain(t, "log", r.log, "::warning::", "::notice::")
	mustContain(t, "gh calls", r.calls, "repos/"+testRepo+"/compare/main..."+reviewSHA)
}

// Earlier comments: only trusted authors are kept, and labels distinguish outdated and file-level ones.
func TestSetupPriorCommentsTrustedOnly(t *testing.T) {
	r := runSetup(t, defaultScenario())
	mustContain(t, "prior-comments.md", r.prior,
		"## comment 1 by owner-user on a.go:3\n",
		"## comment 2 by claude[bot] on a.go:9 [outdated]",
		"## comment 3 by Copilot on b.go:? [file-level]",
		"owner line comment")
	mustNotContain(t, "prior-comments.md", r.prior, "copilot-fan", "drive-by")
}

// A newer commit was pushed before the run started: the file list must come from compare.
func TestSetupHeadAlreadyMoved(t *testing.T) {
	sc := defaultScenario()
	sc.heads = []string{newerSHA}
	r := runSetup(t, sc)
	assertClean(t, r)
	mustContain(t, "log", r.log, "::notice::The PR head ("+newerSHA+") is newer than the reviewed commit")
	mustContain(t, "pr-meta.md", r.meta, "(file list from: compare)", "- compare-only.go")
	mustNotContain(t, "pr-meta.md", r.meta, "pulls-a.go")
}

// The head moves while pulls/N/files is being fetched: that list may mix commits, so use compare.
func TestSetupHeadMovesDuringFetch(t *testing.T) {
	sc := defaultScenario()
	sc.heads = []string{reviewSHA, newerSHA}
	r := runSetup(t, sc)
	assertClean(t, r)
	mustContain(t, "log", r.log, "::notice::The PR head changed while fetching files")
	mustContain(t, "pr-meta.md", r.meta, "(file list from: compare)", "- compare-only.go")
}

// pulls/N fails: gh prints error JSON to stdout, which must not be mistaken for a moved head or
// leak into pr-meta.md.
func TestSetupHeadLookupFails(t *testing.T) {
	sc := defaultScenario()
	sc.fail = []string{"pull"}
	r := runSetup(t, sc)
	assertClean(t, r)
	mustContain(t, "log", r.log,
		"::warning::Could not read the PR head",
		"::warning::Could not fetch title and description")
	mustNotContain(t, "log", r.log, "is newer than the reviewed commit")
	mustContain(t, "pr-meta.md", r.meta, "(could not fetch title and description)", "(file list from: compare)")
}

// pulls/N/files fails: fall back to compare for the file list.
func TestSetupPullFilesFails(t *testing.T) {
	sc := defaultScenario()
	sc.fail = []string{"pull_files"}
	r := runSetup(t, sc)
	assertClean(t, r)
	mustContain(t, "log", r.log, "::warning::Could not fetch pulls/"+testPR+"/files")
	mustContain(t, "pr-meta.md", r.meta, "(file list from: compare)", "- compare-only.go")
}

// GitHub refuses the full diff: it is rebuilt from per-file patches, including renames, deletions and
// files with no patch.
func TestSetupDiffFallback(t *testing.T) {
	sc := defaultScenario()
	sc.fail = []string{"compare_diff"}
	sc.pullFiles = []map[string]any{
		file("kept.go", "modified", nil),
		file("new-name.go", "renamed", map[string]any{"previous_filename": "old-name.go", "patch": nil}),
		file("gone.go", "removed", map[string]any{"patch": nil}),
		file("image.png", "modified", map[string]any{"patch": nil}),
	}
	r := runSetup(t, sc)
	assertClean(t, r)
	mustContain(t, "log", r.log, "::notice::compare diff failed")
	mustContain(t, "pr.diff", r.diff,
		"diff --git a/kept.go b/kept.go", "+kept.go",
		"diff --git a/old-name.go b/new-name.go", "(renamed from old-name.go)", "# (renamed without content changes)",
		"# (file deleted; it is not in the checkout)",
		"# (no patch: binary or too large; Read the file directly)")
	mustNotContain(t, "pr.diff", r.diff, "from-compare.go")
	mustContain(t, "pr-meta.md", r.meta, "- new-name.go (renamed, +1 -0) renamed from old-name.go")
}

// Diff and file list both unavailable: warn, but still produce the other files.
func TestSetupNoDiffAtAll(t *testing.T) {
	sc := defaultScenario()
	sc.fail = []string{"compare_diff", "pull_files", "compare"}
	r := runSetup(t, sc)
	assertClean(t, r)
	mustContain(t, "log", r.log,
		"::warning::Could not fetch the changed-file list.",
		"::warning::Could not build the PR diff",
		"::warning::Could not fetch the commit list")
	mustContain(t, "pr-meta.md", r.meta, "(could not fetch the file list)", "(could not fetch the commit list)")
	// A failed fetch must not be misreported as GitHub truncating the lists.
	mustNotContain(t, "pr-meta.md", r.meta, "NOTE:")
	if r.diff != "" {
		t.Errorf("pr.diff should be empty, got:\n%s", r.diff)
	}
}

// Comments fetch fails: warn and leave an empty file rather than error JSON.
func TestSetupCommentsFail(t *testing.T) {
	sc := defaultScenario()
	sc.fail = []string{"comments"}
	r := runSetup(t, sc)
	assertClean(t, r)
	mustContain(t, "log", r.log, "::warning::Could not fetch earlier inline comments")
	if r.prior != "" {
		t.Errorf("prior-comments.md should be empty, got:\n%s", r.prior)
	}
}

// Fewer commits than total_commits: the reviewer is told the lists may be incomplete.
func TestSetupCommitListShort(t *testing.T) {
	sc := defaultScenario()
	sc.commits, sc.totalCommit = 2, 5
	r := runSetup(t, sc)
	assertClean(t, r)
	mustContain(t, "log", r.log, "::warning::GitHub returned only 2 of 5 commits; the commit list is incomplete.")
	mustContain(t, "pr-meta.md", r.meta, "NOTE: GitHub returned only 2 of 5 commits; the commit list above is incomplete.")
	// Only the commit list is short; the file list must not be called truncated.
	mustNotContain(t, "pr-meta.md", r.meta, "file list (and")
}

// The compare API returns at most 300 files: reaching that cap triggers the NOTE.
func TestSetupCompareFileCap(t *testing.T) {
	sc := defaultScenario()
	sc.heads = []string{newerSHA} // force the compare source
	sc.compareFile = make([]map[string]any, 300)
	for i := range sc.compareFile {
		sc.compareFile[i] = file(fmt.Sprintf("f%03d.go", i), "added", nil)
	}
	r := runSetup(t, sc)
	assertClean(t, r)
	mustContain(t, "log", r.log, "::warning::GitHub returned 300 changed files, the most it returns here")
	mustContain(t, "pr-meta.md", r.meta, "NOTE: GitHub returned 300 changed files, the most it returns here", "may be truncated")
	mustNotContain(t, "pr-meta.md", r.meta, "commit list above is incomplete")
}

// The compare JSON (commits, total_commits) fails: the commit list is reported missing, and the
// truncation check is skipped instead of failing on a non-numeric total.
func TestSetupCompareJSONFails(t *testing.T) {
	sc := defaultScenario()
	sc.fail = []string{"compare"} // compare JSON fails; the diff (compare_diff) still works
	r := runSetup(t, sc)
	assertClean(t, r)
	mustContain(t, "log", r.log, "::warning::Could not fetch the commit list")
	mustNotContain(t, "pr-meta.md", r.meta, "NOTE:")
}

// The job's skip conditions are repeated in the concurrency group key (skipped runs get their own
// group so they can't cancel a real review). Each clause in the job's `if:` must also appear there.
func TestSkipConditionsInSync(t *testing.T) {
	wf := loadWorkflow(t)
	norm := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	jobIf := norm(wf.Jobs[jobName].If)
	group := norm(wf.Concurrency.Group)
	clauses := []string{
		"!github.event.pull_request.draft",
		"(github.event.action != 'edited' || github.event.changes.base)",
		"(github.actor == 'dependabot[bot]' || github.actor == 'claude[bot]' || (!endsWith(github.actor, '[bot]') && github.actor != 'Copilot'))",
	}
	for _, c := range clauses {
		if !strings.Contains(jobIf, c) {
			t.Errorf("job if: is missing clause %q\n(if: %s)\n(update this test if the condition changed on purpose)", c, jobIf)
		}
		if !strings.Contains(group, c) {
			t.Errorf("concurrency group is missing clause %q that the job if: has; keep them in sync\n(group: %s)", c, group)
		}
	}
	// Events that must cancel a running review join the review group, and the job skips them.
	for _, ev := range []string{"converted_to_draft", "closed"} {
		if !strings.Contains(group, "github.event.action == '"+ev+"'") {
			t.Errorf("concurrency group should route %s into the review group", ev)
		}
	}
	if !strings.Contains(jobIf, "github.event.action != 'closed'") {
		t.Errorf("job if: should skip closed events")
	}
}

// Every LacyLights repo carries an identical copy of the workflow. When sibling repos are checked out
// next to this one (the usual local layout), compare their origin/main copies with this one. Skipped
// in CI, where the siblings aren't present.
func TestWorkflowIdenticalAcrossRepos(t *testing.T) {
	ours, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	siblings := []string{"lacylights-go", "lacylights-fe", "lacylights-mcp", "lacylights-rpi",
		"lacylights-terraform", "lacylights-mac"}
	checked := 0
	for _, repo := range siblings {
		dir := filepath.Join(root, repo)
		if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
			continue
		}
		theirs, err := exec.Command("git", "-C", dir, "show",
			"origin/main:.github/workflows/claude-inline-review.yml").Output()
		if err != nil {
			t.Logf("%s: no claude-inline-review.yml on origin/main yet; skipping", repo)
			continue
		}
		checked++
		if string(theirs) != string(ours) {
			t.Errorf("%s: origin/main copy of claude-inline-review.yml differs from lacylights-test's; "+
				"the workflow must be identical in every repo", repo)
		}
	}
	if checked == 0 {
		t.Skip("no sibling repo checkouts with the workflow on origin/main found next to lacylights-test")
	}
}
