// Package inlinereview tests the shared .github/workflows/claude-inline-review.yml workflow, which is
// copied unchanged into every LacyLights repository.
//
// The workflow's "Collect PR context" step is inline shell. It has to stay inline: a script file in
// the repository could be changed by a pull request and would run on the same runner before the step
// that holds the Claude token. These tests therefore pull the step's script out of the workflow YAML
// and run it with bash, using a fake `gh` and canned GitHub API data. They also check that the
// workflow's duplicated skip conditions stay in sync and that its reviewer sandbox is unchanged. An
// opt-in check (`make check-workflow-drift`) compares origin/main copies: each known repository checked
// out next to this one must carry an identical copy, other lacylights checkouts are compared if they have
// one, and repositories that aren't checked out are listed as not checked.
package inlinereview
