// Package inlinereview tests the shared .github/workflows/claude-inline-review.yml workflow, which is
// copied unchanged into every LacyLights repository.
//
// The workflow's "Collect PR context" step is inline shell. It has to stay inline: a script file in
// the repository could be changed by a pull request and would run on the same runner before the step
// that holds the Claude token. These tests therefore pull the step's script out of the workflow YAML
// and run it with bash, using a fake `gh` and canned GitHub API data. They also check that the
// workflow's duplicated skip conditions stay in sync, and that sibling repositories checked out next
// to this one carry an identical copy of the workflow.
package inlinereview
