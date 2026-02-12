# Publishing promptctl to Homebrew

## Overview

You need 3 things:
1. A GitHub repo for the source code (`promptctl`)
2. A separate GitHub repo for the Homebrew tap (`homebrew-tap`)
3. A GitHub Personal Access Token so GoReleaser can push the formula

GoReleaser + GitHub Actions handles everything automatically after initial setup.
When you push a version tag, it builds binaries, creates a GitHub Release,
and updates the Homebrew formula in your tap repo.

## Step-by-Step

### 1. Create the source repo

```bash
cd promptctl
git init
git add .
git commit -m "feat: initial release of promptctl"
gh repo create promptctl --public --source=. --push
```

### 2. Create the tap repo

```bash
# The repo MUST be named "homebrew-tap" (or "homebrew-<something>")
# This naming convention is required by Homebrew
gh repo create homebrew-tap --public --description "Homebrew formulae"

# Or manually: https://github.com/new -> name it "homebrew-tap"
```

### 3. Create a GitHub Personal Access Token

Go to: https://github.com/settings/tokens?type=beta

Create a fine-grained token with:
- **Name:** goreleaser-homebrew
- **Expiration:** 90 days (or longer)
- **Repository access:** Select "homebrew-tap" repo only
- **Permissions:** Contents -> Read and Write

Copy the token.

### 4. Add the token as a repo secret

Go to your `promptctl` repo -> Settings -> Secrets and variables -> Actions

Add a new secret:
- **Name:** `HOMEBREW_TAP_GITHUB_TOKEN`
- **Value:** (paste the token from step 3)

### 5. Update .goreleaser.yaml

Edit `.goreleaser.yaml` and replace `YOUR_GITHUB_USERNAME` with your actual
GitHub username in both the `owner` field and the `homepage` URL.

### 6. Tag and release

```bash
# Make sure everything is committed
git add .
git commit -m "ci: add goreleaser and github actions"

# Create a version tag
git tag -a v0.1.0 -m "Initial release"

# Push code and tag
git push origin main
git push origin v0.1.0
```

GitHub Actions will automatically:
1. Build binaries for macOS (Intel + Apple Silicon), Linux, Windows
2. Create a GitHub Release with changelog
3. Push a Homebrew formula to your `homebrew-tap` repo

### 7. Verify

After the action completes (~2 min), check:
- GitHub Release: https://github.com/YOUR_USERNAME/promptctl/releases
- Formula: https://github.com/YOUR_USERNAME/homebrew-tap/tree/main/Formula

### 8. Install!

Now anyone (including you) can install with:

```bash
brew tap YOUR_USERNAME/tap
brew install promptctl
```

## Updating

Every future release is just:

```bash
git tag -a v0.2.0 -m "Add new feature X"
git push origin v0.2.0
```

GoReleaser handles everything else. Users update with `brew upgrade promptctl`.

## Troubleshooting

**Action fails with permission error on tap repo:**
Your HOMEBREW_TAP_GITHUB_TOKEN doesn't have write access to homebrew-tap.
Regenerate with correct repo permissions.

**Formula not found after `brew tap`:**
Check that your tap repo has `Formula/promptctl.rb` (not in root).

**`brew install` downloads wrong binary:**
GoReleaser auto-detects OS/arch. If issues persist, check the
`archives.name_template` in `.goreleaser.yaml`.

**Want to test locally before pushing:**
```bash
# Install goreleaser
brew install goreleaser

# Dry run (builds but doesn't publish)
goreleaser release --snapshot --clean
```
