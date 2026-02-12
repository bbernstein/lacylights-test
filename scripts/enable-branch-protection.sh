#!/bin/bash

# Enable branch protection on all LacyLights repositories
# This ensures CI checks must pass before merging to main

set -e

echo "🔒 LacyLights Branch Protection Setup"
echo "===================================="
echo ""
echo "This script will enable branch protection on all LacyLights repos."
echo "Required checks will be enforced before merging to main."
echo ""

# Check if gh is installed
if ! command -v gh &> /dev/null; then
    echo "❌ Error: GitHub CLI (gh) is not installed."
    echo "Install with: brew install gh"
    exit 1
fi

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo "❌ Error: Not authenticated with GitHub."
    echo "Run: gh auth login"
    exit 1
fi

echo "✅ GitHub CLI is installed and authenticated"
echo ""

# Function to enable branch protection
enable_protection() {
    local repo=$1
    local checks=$2

    echo "📝 Configuring $repo..."

    # Create JSON payload
    local payload=$(cat <<EOF
{
  "required_status_checks": {
    "strict": true,
    "contexts": [$checks]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "required_approving_review_count": 1
  },
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false
}
EOF
)

    # Apply protection
    local error_output
    if error_output=$(gh api -X PUT "repos/$repo/branches/main/protection" --input - <<< "$payload" 2>&1); then
        echo "   ✅ Branch protection enabled"
    else
        echo "   ⚠️  Failed to enable branch protection"
        echo "   Error: $error_output"
    fi
}

# Configure each repository
echo "Configuring repositories..."
echo ""

enable_protection "bbernstein/lacylights-test" '"test", "lint", "e2e"'
enable_protection "bbernstein/lacylights-go" '"test", "lint", "build"'
enable_protection "bbernstein/lacylights-fe" '"lint-and-type-check", "test", "security-audit"'
enable_protection "bbernstein/lacylights-mcp" '"lint-and-build", "security-audit", "mcp-validation"'

echo ""
echo "✅ Branch protection configuration complete!"
echo ""
echo "Verification:"
gh api repos/bbernstein/lacylights-test/branches/main/protection 2>&1 | \
  grep -q "required_status_checks" && \
  echo "  ✅ lacylights-test protected" || \
  echo "  ⚠️  lacylights-test may not be protected"

gh api repos/bbernstein/lacylights-go/branches/main/protection 2>&1 | \
  grep -q "required_status_checks" && \
  echo "  ✅ lacylights-go protected" || \
  echo "  ⚠️  lacylights-go may not be protected"

gh api repos/bbernstein/lacylights-fe/branches/main/protection 2>&1 | \
  grep -q "required_status_checks" && \
  echo "  ✅ lacylights-fe protected" || \
  echo "  ⚠️  lacylights-fe may not be protected"

gh api repos/bbernstein/lacylights-mcp/branches/main/protection 2>&1 | \
  grep -q "required_status_checks" && \
  echo "  ✅ lacylights-mcp protected" || \
  echo "  ⚠️  lacylights-mcp may not be protected"

echo ""
echo "Note: From now on, all PRs must pass required CI checks before merging."
echo "This prevents accidentally merging code with test failures."
