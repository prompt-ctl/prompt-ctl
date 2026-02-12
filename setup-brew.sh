#!/usr/bin/env bash
# ============================================================================
# setup-brew.sh - One-shot script to publish promptctl to Homebrew
#
# What this does:
#   1. Validates prerequisites (gh CLI, Go, git)
#   2. Detects or asks for your GitHub username
#   3. Creates the "homebrew-tap" repo on GitHub (if it doesn't exist)
#   4. Patches .goreleaser.yaml with your actual username
#   5. Creates a fine-grained GitHub token (opens browser for you)
#   6. Stores the token as a repo secret (HOMEBREW_TAP_GITHUB_TOKEN)
#   7. Initializes git, pushes code, tags v0.1.0, and triggers the release
#
# Usage:
#   chmod +x setup-brew.sh
#   ./setup-brew.sh
#
# After this script completes, anyone can install with:
#   brew tap <your-username>/tap
#   brew install promptctl
# ============================================================================

set -euo pipefail

# -- Colors for output -------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[OK]${NC} $1"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
error()   { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }
step()    { echo -e "\n${BOLD}==> Step $1: $2${NC}"; }

# -- Step 0: Preflight checks ------------------------------------------------
step "0" "Checking prerequisites"

# Check for gh CLI - this is the backbone of the entire automation
if ! command -v gh &>/dev/null; then
    error "GitHub CLI (gh) not found. Install it first:
    brew install gh
    Then authenticate: gh auth login"
fi

# Verify gh is authenticated (otherwise every API call will fail)
if ! gh auth status &>/dev/null 2>&1; then
    error "GitHub CLI not authenticated. Run: gh auth login"
fi
success "GitHub CLI authenticated"

# Check for Go (needed to verify the project builds)
if ! command -v go &>/dev/null; then
    error "Go not found. Install it: brew install go"
fi
success "Go $(go version | awk '{print $3}') found"

# Check for git
if ! command -v git &>/dev/null; then
    error "git not found. Install it: brew install git"
fi
success "git found"

# Make sure we're in the promptctl project directory
if [[ ! -f "main.go" ]] || [[ ! -f ".goreleaser.yaml" ]]; then
    error "Run this script from the promptctl project root (where main.go is)"
fi
success "Running from project root"

# -- Step 1: Detect GitHub username ------------------------------------------
step "1" "Detecting GitHub identity"

# gh knows who you are - let's use that instead of asking
GITHUB_USERNAME=$(gh api user --jq '.login' 2>/dev/null || echo "")

if [[ -z "$GITHUB_USERNAME" ]]; then
    echo -n "Could not detect GitHub username. Enter it manually: "
    read -r GITHUB_USERNAME
fi

if [[ -z "$GITHUB_USERNAME" ]]; then
    error "GitHub username is required"
fi
success "GitHub username: $GITHUB_USERNAME"

# -- Step 2: Verify the project builds ---------------------------------------
step "2" "Verifying project builds"

if ! go build -o /dev/null . 2>&1; then
    error "Project failed to compile. Fix build errors first."
fi
success "Project compiles cleanly"

# -- Step 3: Create the source repo on GitHub ---------------------------------
step "3" "Setting up source repo (promptctl)"

# Check if origin remote already exists and points to GitHub
EXISTING_REMOTE=$(git remote get-url origin 2>/dev/null || echo "")

if [[ -n "$EXISTING_REMOTE" ]] && [[ "$EXISTING_REMOTE" == *"github.com"* ]]; then
    success "Source repo already configured: $EXISTING_REMOTE"
else
    # Check if repo exists on GitHub already
    if gh repo view "$GITHUB_USERNAME/promptctl" &>/dev/null 2>&1; then
        info "Repo exists on GitHub but no local remote. Adding it."
        git remote add origin "https://github.com/$GITHUB_USERNAME/promptctl.git" 2>/dev/null || \
            git remote set-url origin "https://github.com/$GITHUB_USERNAME/promptctl.git"
    else
        info "Creating promptctl repo on GitHub..."
        # Initialize git if not already done
        if [[ ! -d ".git" ]]; then
            git init
            git add .
            git commit -m "feat: initial release of promptctl"
        fi
        gh repo create promptctl --public \
            --description "CLI toolkit that transforms raw intent into structured, optimized prompts" \
            --source=. --push
    fi
    success "Source repo ready"
fi

# -- Step 4: Create the Homebrew tap repo -------------------------------------
step "4" "Setting up Homebrew tap repo (homebrew-tap)"

if gh repo view "$GITHUB_USERNAME/homebrew-tap" &>/dev/null 2>&1; then
    success "homebrew-tap repo already exists"
else
    info "Creating homebrew-tap repo on GitHub..."
    gh repo create homebrew-tap --public \
        --description "Homebrew formulae for $GITHUB_USERNAME's tools"
    
    # Homebrew tap repos need at least one commit, otherwise brew tap fails.
    # We create a minimal README so the repo isn't empty.
    TMPDIR=$(mktemp -d)
    cd "$TMPDIR"
    git init
    echo "# Homebrew Tap" > README.md
    echo "" >> README.md
    echo "Install formulae from this tap:" >> README.md
    echo "" >> README.md
    echo '```bash' >> README.md
    echo "brew tap $GITHUB_USERNAME/tap" >> README.md
    echo "brew install promptctl" >> README.md
    echo '```' >> README.md
    mkdir -p Formula
    touch Formula/.gitkeep
    git add .
    git commit -m "init: homebrew tap repo"
    git branch -M main
    git remote add origin "https://github.com/$GITHUB_USERNAME/homebrew-tap.git"
    git push -u origin main
    cd - > /dev/null
    rm -rf "$TMPDIR"
    
    success "homebrew-tap repo created and initialized"
fi

# -- Step 5: Patch .goreleaser.yaml with actual username ----------------------
step "5" "Configuring GoReleaser with your GitHub identity"

# Replace the placeholder username in .goreleaser.yaml
# Using a portable sed approach that works on both macOS and Linux
if grep -q "YOUR_GITHUB_USERNAME" .goreleaser.yaml; then
    if [[ "$(uname)" == "Darwin" ]]; then
        # macOS sed requires an explicit empty string for -i
        sed -i '' "s/YOUR_GITHUB_USERNAME/$GITHUB_USERNAME/g" .goreleaser.yaml
    else
        sed -i "s/YOUR_GITHUB_USERNAME/$GITHUB_USERNAME/g" .goreleaser.yaml
    fi
    success "Patched .goreleaser.yaml with username: $GITHUB_USERNAME"
else
    success ".goreleaser.yaml already configured"
fi

# Also update the go.mod module path to match the actual repo
if grep -q "github.com/oleg/promptctl" go.mod; then
    if [[ "$(uname)" == "Darwin" ]]; then
        # Update module path in go.mod and all Go source files
        find . -name "*.go" -exec sed -i '' "s|github.com/oleg/promptctl|github.com/$GITHUB_USERNAME/promptctl|g" {} +
        sed -i '' "s|github.com/oleg/promptctl|github.com/$GITHUB_USERNAME/promptctl|g" go.mod
    else
        find . -name "*.go" -exec sed -i "s|github.com/oleg/promptctl|github.com/$GITHUB_USERNAME/promptctl|g" {} +
        sed -i "s|github.com/oleg/promptctl|github.com/$GITHUB_USERNAME/promptctl|g" go.mod
    fi
    success "Updated Go module path to github.com/$GITHUB_USERNAME/promptctl"
fi

# -- Step 6: Create GitHub token and store as secret --------------------------
step "6" "Setting up GitHub token for Homebrew tap access"

# Check if the secret already exists. If it does, skip this step entirely
# because recreating it would invalidate the old one.
EXISTING_SECRET=$(gh secret list --repo "$GITHUB_USERNAME/promptctl" 2>/dev/null | grep "HOMEBREW_TAP_GITHUB_TOKEN" || echo "")

if [[ -n "$EXISTING_SECRET" ]]; then
    success "HOMEBREW_TAP_GITHUB_TOKEN secret already exists"
else
    info "We need a GitHub token that lets GoReleaser push formulas to homebrew-tap."
    info "The easiest way is to create one via the gh CLI."
    echo ""
    
    # Try to create a token programmatically first. If that fails (e.g., the user
    # has SSO or fine-grained tokens aren't supported via CLI), fall back to
    # manual creation with a browser link.
    
    # Method: ask the user to create a classic token with repo scope.
    # Fine-grained tokens are better but harder to automate via CLI.
    echo -e "${YELLOW}I need to open GitHub in your browser to create a Personal Access Token.${NC}"
    echo -e "${YELLOW}When the page opens:${NC}"
    echo "  1. Name it: goreleaser-homebrew-tap"
    echo "  2. Set expiration to 90 days (or longer)"
    echo "  3. Under 'Repository access', select 'Only select repositories'"
    echo "     and choose: homebrew-tap"
    echo "  4. Under 'Permissions' -> 'Repository permissions':"
    echo "     set 'Contents' to 'Read and write'"
    echo "  5. Click 'Generate token' and copy it"
    echo ""
    echo -n "Press Enter to open GitHub in your browser..."
    read -r
    
    # Open the token creation page
    open "https://github.com/settings/personal-access-tokens/new" 2>/dev/null || \
        xdg-open "https://github.com/settings/personal-access-tokens/new" 2>/dev/null || \
        warn "Could not open browser. Go to: https://github.com/settings/personal-access-tokens/new"
    
    echo ""
    echo -n "Paste the token here (input is hidden): "
    read -rs HOMEBREW_TOKEN
    echo ""
    
    if [[ -z "$HOMEBREW_TOKEN" ]]; then
        error "Token cannot be empty. Re-run the script when you have the token."
    fi
    
    # Store the token as a GitHub Actions secret on the source repo.
    # This is what the release workflow references as secrets.HOMEBREW_TAP_GITHUB_TOKEN
    echo "$HOMEBREW_TOKEN" | gh secret set HOMEBREW_TAP_GITHUB_TOKEN \
        --repo "$GITHUB_USERNAME/promptctl"
    
    success "Token stored as repo secret: HOMEBREW_TAP_GITHUB_TOKEN"
fi

# -- Step 7: Commit everything and push --------------------------------------
step "7" "Committing and pushing code"

# Make sure git is initialized and configured
if [[ ! -d ".git" ]]; then
    git init
fi

# Set default branch to main
git branch -M main 2>/dev/null || true

# Stage all files
git add -A

# Only commit if there are changes (the script might be re-run)
if ! git diff --cached --quiet 2>/dev/null; then
    git commit -m "feat: initial release of promptctl

- CLI toolkit for prompt engineering
- Template system with variable substitution
- 'create' command for intent-to-prompt transformation
- 5 starter templates (review, debug, arch, commit, explain)
- GoReleaser + GitHub Actions for automated releases
- Homebrew tap support"
    success "Changes committed"
else
    success "Nothing new to commit"
fi

# Push to GitHub. Set upstream if not already configured.
if git remote get-url origin &>/dev/null; then
    git push -u origin main 2>/dev/null || git push origin main
    success "Code pushed to GitHub"
else
    error "No git remote 'origin' configured. Something went wrong in step 3."
fi

# -- Step 8: Tag and release --------------------------------------------------
step "8" "Tagging v0.1.0 and triggering release"

# Check if v0.1.0 already exists (don't double-tag)
if git tag -l "v0.1.0" | grep -q "v0.1.0"; then
    warn "Tag v0.1.0 already exists. Checking if we need a new version..."
    
    # Find the latest tag and bump it
    LATEST_TAG=$(git tag -l "v*" --sort=-v:refname | head -1)
    if [[ -n "$LATEST_TAG" ]]; then
        # Simple bump: extract minor version and increment
        MAJOR=$(echo "$LATEST_TAG" | sed 's/v//' | cut -d. -f1)
        MINOR=$(echo "$LATEST_TAG" | sed 's/v//' | cut -d. -f2)
        PATCH=$(echo "$LATEST_TAG" | sed 's/v//' | cut -d. -f3)
        NEW_PATCH=$((PATCH + 1))
        NEW_TAG="v${MAJOR}.${MINOR}.${NEW_PATCH}"
        
        info "Latest tag is $LATEST_TAG. Creating $NEW_TAG"
        git tag -a "$NEW_TAG" -m "Release $NEW_TAG"
        git push origin "$NEW_TAG"
        success "Pushed tag $NEW_TAG - release workflow triggered"
    fi
else
    git tag -a v0.1.0 -m "Initial release of promptctl"
    git push origin v0.1.0
    success "Pushed tag v0.1.0 - release workflow triggered"
fi

# -- Step 9: Wait for release and verify -------------------------------------
step "9" "Waiting for GitHub Actions to build the release"

info "The release workflow typically takes 1-3 minutes."
info "Monitoring progress..."
echo ""

# Poll the Actions run status. We look for the most recent workflow run
# triggered by a tag push and wait until it completes.
MAX_WAIT=300  # 5 minutes max
WAITED=0
INTERVAL=10

while [[ $WAITED -lt $MAX_WAIT ]]; do
    # Get the latest workflow run status
    RUN_STATUS=$(gh run list --repo "$GITHUB_USERNAME/promptctl" \
        --workflow=release.yml --limit=1 --json status,conclusion \
        --jq '.[0] | "\(.status) \(.conclusion)"' 2>/dev/null || echo "unknown")
    
    STATUS=$(echo "$RUN_STATUS" | awk '{print $1}')
    CONCLUSION=$(echo "$RUN_STATUS" | awk '{print $2}')
    
    if [[ "$STATUS" == "completed" ]]; then
        if [[ "$CONCLUSION" == "success" ]]; then
            success "Release workflow completed successfully!"
            break
        else
            error "Release workflow failed with conclusion: $CONCLUSION
Check the logs: gh run list --repo $GITHUB_USERNAME/promptctl --workflow=release.yml
Or visit: https://github.com/$GITHUB_USERNAME/promptctl/actions"
        fi
    elif [[ "$STATUS" == "unknown" ]]; then
        # Workflow might not have started yet
        echo -ne "\r  Waiting for workflow to start... (${WAITED}s)"
    else
        echo -ne "\r  Workflow status: $STATUS (${WAITED}s elapsed)    "
    fi
    
    sleep $INTERVAL
    WAITED=$((WAITED + INTERVAL))
done

echo "" # newline after the progress indicator

if [[ $WAITED -ge $MAX_WAIT ]]; then
    warn "Timed out waiting for the workflow. It may still be running."
    warn "Check: https://github.com/$GITHUB_USERNAME/promptctl/actions"
    warn "Once it completes, the install commands below will work."
fi

# -- Done! -------------------------------------------------------------------
echo ""
echo -e "${GREEN}${BOLD}============================================================${NC}"
echo -e "${GREEN}${BOLD}  promptctl is published to Homebrew!${NC}"
echo -e "${GREEN}${BOLD}============================================================${NC}"
echo ""
echo -e "  ${BOLD}Install:${NC}"
echo -e "    brew tap $GITHUB_USERNAME/tap"
echo -e "    brew install promptctl"
echo ""
echo -e "  ${BOLD}Verify:${NC}"
echo -e "    promptctl version"
echo -e "    promptctl init"
echo -e "    promptctl create \"your idea here\""
echo ""
echo -e "  ${BOLD}Future releases:${NC}"
echo -e "    git tag -a v0.2.0 -m \"description\""
echo -e "    git push origin v0.2.0"
echo -e "    # That's it. GoReleaser handles the rest."
echo ""
echo -e "  ${BOLD}Links:${NC}"
echo -e "    Source:   https://github.com/$GITHUB_USERNAME/promptctl"
echo -e "    Releases: https://github.com/$GITHUB_USERNAME/promptctl/releases"
echo -e "    Tap:      https://github.com/$GITHUB_USERNAME/homebrew-tap"
echo ""
