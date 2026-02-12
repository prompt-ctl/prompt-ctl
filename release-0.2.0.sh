#!/usr/bin/env bash
# ============================================================================
# release-0.2.0.sh - One-time release script for promptctl v0.2.0
#
# Run from the promptctl project root:
#   chmod +x release-0.2.0.sh
#   ./release-0.2.0.sh
# ============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[OK]${NC} $1"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $1"; }
error()   { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

VERSION="v0.2.0"

echo -e "\n${BOLD}Releasing promptctl ${VERSION}${NC}\n"

# ── Preflight ──────────────────────────────────────────────────────
info "Running preflight checks..."

[[ -f "main.go" ]]          || error "Not in project root (no main.go)"
[[ -f ".goreleaser.yaml" ]] || error "Missing .goreleaser.yaml"
command -v go &>/dev/null    || error "Go not installed"
command -v gh &>/dev/null    || error "GitHub CLI not installed"
gh auth status &>/dev/null 2>&1 || error "GitHub CLI not authenticated"

# Verify it builds
go build -o /dev/null . 2>&1 || error "Build failed. Fix errors first."
success "Build clean"

# Check we're on the right branch
BRANCH=$(git branch --show-current)
if [[ "$BRANCH" != "main" ]]; then
    warn "On branch '$BRANCH', not 'main'. Continue anyway? (y/N)"
    read -r CONFIRM
    [[ "$CONFIRM" == "y" ]] || exit 0
fi

# Check for uncommitted changes
if ! git diff --quiet || ! git diff --cached --quiet; then
    info "Staging and committing all changes..."
    git add -A
    git commit -m "feat(v0.2.0): LLM integration, interactive onboarding, cost estimation

NEW FEATURES:
- Interactive setup wizard: 'promptctl config' walks through provider,
  model, and API key setup in 4 guided steps
- Multi-provider support: Anthropic, OpenAI, Groq, DeepSeek with
  10 models and live pricing
- Cost estimation: 'promptctl cost' shows per-call costs before sending
- Cost comparison: 'promptctl cost --compare' compares all models
- Direct LLM execution: 'promptctl send' renders + sends in one call
  with real-time cost reporting after each call
- Model switcher: 'promptctl models --set' for interactive model selection
- API key management: stored securely in ~/.promptctl/llm.json (0600)
- Landing page for GitHub Pages
- Polished Homebrew tap README

IMPROVEMENTS:
- Help text reorganized into logical sections
- Every 'send' shows savings vs unstructured prompting
- Flag-based config still works for scripting/CI"
    success "Changes committed"
else
    success "Working tree clean"
fi

# Push code first
info "Pushing to origin..."
git push origin "$BRANCH" 2>/dev/null || git push -u origin "$BRANCH"
success "Code pushed"

# Check if tag already exists
if git tag -l "$VERSION" | grep -q "$VERSION"; then
    error "Tag $VERSION already exists. Use a different version."
fi

# ── Tag and release ────────────────────────────────────────────────
info "Creating tag $VERSION..."
git tag -a "$VERSION" -m "Release $VERSION - LLM Integration & Cost Estimation

## What's New

### Interactive Setup Wizard
Run 'promptctl config' and follow 4 simple steps:
1. Choose provider (Anthropic, OpenAI, Groq, DeepSeek)
2. Pick your model from the provider's lineup
3. Create and paste your API key (browser opens automatically)
4. Confirmation with quick-start commands

### Multi-Provider LLM Support
- 4 providers, 10 models with current pricing
- Direct execution: 'promptctl send review --file=auth.ts'
- Cost report after every call showing actual spend

### Cost Estimation
- 'promptctl cost review --file=main.go' - estimate before sending
- 'promptctl cost --compare \"your idea\"' - compare all 10 models
- Every call shows savings vs unstructured prompting (~67%)

### Model Switching
- 'promptctl models' - see all models with pricing
- 'promptctl models --set' - interactive model picker

### Landing Page
- GitHub Pages site with animated terminal demo
- Auto-deploys from docs/ folder"

git push origin "$VERSION"
success "Tag $VERSION pushed"

# ── Monitor release ────────────────────────────────────────────────
info "Waiting for GitHub Actions release workflow..."
echo ""

GITHUB_USERNAME=$(gh api user --jq '.login' 2>/dev/null)
MAX_WAIT=300
WAITED=0

while [[ $WAITED -lt $MAX_WAIT ]]; do
    RUN_STATUS=$(gh run list --repo "$GITHUB_USERNAME/promptctl" \
        --workflow=release.yml --limit=1 --json status,conclusion \
        --jq '.[0] | "\(.status) \(.conclusion)"' 2>/dev/null || echo "unknown")
    
    STATUS=$(echo "$RUN_STATUS" | awk '{print $1}')
    CONCLUSION=$(echo "$RUN_STATUS" | awk '{print $2}')
    
    if [[ "$STATUS" == "completed" ]]; then
        if [[ "$CONCLUSION" == "success" ]]; then
            echo ""
            success "Release workflow completed!"
            break
        else
            echo ""
            error "Release failed. Check: https://github.com/$GITHUB_USERNAME/promptctl/actions"
        fi
    fi
    
    echo -ne "\r  ⏳ $STATUS (${WAITED}s)   "
    sleep 10
    WAITED=$((WAITED + 10))
done

if [[ $WAITED -ge $MAX_WAIT ]]; then
    echo ""
    warn "Timed out. Check: https://github.com/$GITHUB_USERNAME/promptctl/actions"
fi

# ── Done ───────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}${BOLD}════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}${BOLD}  promptctl $VERSION released!${NC}"
echo -e "${GREEN}${BOLD}════════════════════════════════════════════════════════${NC}"
echo ""
echo "  Update locally:  brew upgrade promptctl"
echo ""
echo "  Users install:   brew tap $GITHUB_USERNAME/tap"
echo "                   brew install promptctl"
echo "                   promptctl config"
echo ""
echo "  Links:"
echo "    Release:  https://github.com/$GITHUB_USERNAME/promptctl/releases/tag/$VERSION"
echo "    Website:  https://$GITHUB_USERNAME.github.io/promptctl"
echo ""
