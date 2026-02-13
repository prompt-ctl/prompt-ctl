# Release setup (one-time)

The main repo is **private**; Homebrew install works for everyone because binaries are published to a **public** repo and the cask points there.

## One-time steps (with gh CLI)

From the promptctl repo root:

```bash
./scripts/setup-public-releases-gh.sh
```

This will:

1. **Create the public repo** `oleg-koval/promptctl-releases` (if it doesn’t exist):
   - `gh repo create oleg-koval/promptctl-releases --public --description "..."`

2. **Set secret** `RELEASES_REPO_GITHUB_TOKEN` in this repo:
   - Prompts for a PAT with `repo` scope, or use `export RELEASES_REPO_GITHUB_TOKEN=ghp_...` before running.

3. **Optionally set** `HOMEBREW_TAP_GITHUB_TOKEN` (write access to `oleg-koval/homebrew-tap`):
   - Same script will prompt or use env if the secret isn’t set yet.

**Set secrets with gh only** (after creating a PAT at https://github.com/settings/tokens/new with `repo` scope):

```bash
# From promptctl repo root. Paste token when prompted, or use stdin:
echo -n "ghp_xxxx" | gh secret set RELEASES_REPO_GITHUB_TOKEN
echo -n "ghp_xxxx" | gh secret set HOMEBREW_TAP_GITHUB_TOKEN   # if not set yet
```

Or prompt for value (no token in shell history): `gh secret set RELEASES_REPO_GITHUB_TOKEN`

Manual alternative (no gh): create the repo and secrets in GitHub → Settings → Secrets and variables → Actions.

## Flow on tag push

1. GoReleaser builds binaries and creates a release on the **private** promptctl repo.
2. Workflow copies those assets to a release on **oleg-koval/promptctl-releases** (public).
3. Workflow pushes the generated cask to **oleg-koval/homebrew-tap**; the cask’s download URL points at promptctl-releases.

Result: anyone can `brew tap oleg-koval/tap && brew install promptctl` with no token.
