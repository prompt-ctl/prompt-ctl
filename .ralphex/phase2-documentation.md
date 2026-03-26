# Promptctl Phase 2: Documentation Generation

You are creating comprehensive documentation for the open-source release of promptctl, a Go CLI tool for prompt engineering templates.

## Context
- Repo: /Users/olegkoval/projects/personal/active/prompt-ctl.com/promptctl
- Codebase status: Fully audited and cleaned (Phase 1 complete)
- Goal: Generate production-ready documentation before public launch
- Target: Developers unfamiliar with promptctl should be able to install and use it in 5 minutes

## Tasks

### 1. Update README.md (Critical - Most Important)
Create a killer README that sells the vision and enables quick start.

**Structure:**
- [x] One-liner: "promptctl is a CLI for version control and testing of LLM prompt templates"
- [x] Problem statement: "Copy-paste prompts are unmaintainable. promptctl treats them like code."
- [x] Quick comparison table: promptctl vs alternatives (vs. LangChain, vs. Prompt.io, vs. PromptEngineering)
- [x] Installation section:
  - [x] Linux: apt, yum, pacman, snap, direct binary, go install
  - [x] macOS: brew, direct binary, go install
  - [x] Windows: direct binary, go install
  - [x] Include simple `curl | bash` one-liner if feasible
- [x] Quick start (5-min runnable example):
  - [x] `promptctl init` explanation
  - [x] Show a template YAML file
  - [x] Show how to use it: `promptctl review --file=src/auth.ts`
  - [x] Show actual output (fake realistic output if needed)
- [x] Feature showcase (with examples):
  - [x] Review templates (code review)
  - [x] Debug templates (error context)
  - [x] Architecture templates (design decisions)
  - [x] Commit templates (changelog generation)
  - [x] Custom templates (extensibility)
- [x] Roadmap section:
  - [x] v1.0.0 features (current)
  - [x] v1.1.0: prompt testing framework
  - [x] v2.0.0: cloud registry (paid)
  - [x] Open core strategy hint: "Core CLI is forever free. Cloud features coming soon."
- [x] Contributing section (link to CONTRIBUTING.md)
- [x] License (Apache 2.0)
- [x] Links: Website, Issues, Discussions

### 2. Create docs/CONTRIBUTING.md
Guide for developers who want to contribute.

**Sections:**
- [x] Developer setup
  - [x] Go version requirement (1.22+)
  - [x] Clone and setup: git clone, cd promptctl, go build -o promptctl .
  - [x] Dependencies: go mod download, go mod verify
- [x] Running the project
  - [x] Build: `go build -o promptctl .`
  - [x] Run: `./promptctl --help`
  - [x] Tests: `go test ./...`
  - [x] Coverage: `go test -cover ./...`
- [x] Code style
  - [x] gofmt: run before commit
  - [x] go vet: must pass
  - [x] Naming conventions (CamelCase, unexported prefix)
  - [x] Error handling best practices
- [x] PR process
  - [x] Fork repo, create feature branch
  - [x] Commit message format: "feat: ...", "fix: ...", "chore: ..."
  - [x] Tests required for new features
  - [x] PR description should reference issue
  - [x] CI must pass before merge
- [x] Testing requirements
  - [x] Unit tests for new functions
  - [x] Integration tests for CLI commands
  - [x] Example: how to test a new command
- [x] Adding a new command
  - [x] Create file in cmd/ (e.g., cmd/mycmd.go)
  - [x] Implement Execute() method
  - [x] Add tests in cmd/mycmd_test.go
  - [x] Update root.go to register command
  - [x] Add to README quick reference
- [x] Reporting issues
  - [x] Bug report template
  - [x] Feature request template
  - [x] Please include: OS, Go version, promptctl version

### 3. Create docs/INSTALL.md
Platform-specific installation instructions.

**Sections:**
- [x] Quick install (copy-paste for lazy users):
  - [x] macOS: `brew install prompt-ctl/tap/promptctl`
  - [x] Linux: `snap install promptctl`
  - [x] Windows: Direct binary download or `choco install promptctl`
  - [x] Universal: `go install github.com/prompt-ctl/prompt-ctl@latest`
- [x] Linux detailed:
  - [x] Ubuntu/Debian: APT repository or direct .deb download
  - [x] Fedora/RHEL: yum/dnf repository
  - [x] Arch Linux: pacman or AUR
  - [x] Alpine: apk (if available)
  - [x] Snap: universal snap package
  - [x] Direct binary: .tar.gz download and PATH setup
  - [x] Verify: `promptctl version`
- [x] macOS detailed:
  - [x] Homebrew: `brew tap prompt-ctl/tap` + `brew install promptctl`
  - [x] Direct binary: download from GitHub releases
  - [x] Apple Silicon note: Built for both Intel and ARM
  - [x] Verify: `promptctl version`
- [x] Windows detailed:
  - [x] Direct binary: download .zip from GitHub releases
  - [x] Add to PATH: environment variable setup
  - [x] Chocolatey (if available): `choco install promptctl`
  - [x] WSL: Use Linux instructions
  - [x] Verify: `promptctl version`
- [x] From source:
  - [x] Requirements: Go 1.22+, git
  - [x] Build: `git clone ... && cd prompt-ctl && go build -o promptctl . && sudo mv promptctl /usr/local/bin/`
  - [x] Verify: `promptctl version`
- [x] Uninstall:
  - [x] macOS/Linux: `brew uninstall promptctl` or `rm /usr/local/bin/promptctl`
  - [x] Linux snap: `snap remove promptctl`
  - [x] Windows: Use Control Panel or delete binary

### 4. Create docs/ARCHITECTURE.md
Deep dive for contributors and curious users.

**Sections:**
- [x] Overview diagram (ASCII art):
  ```
  promptctl CLI
    ├── cmd/ (commands: review, fix, debug, arch, score, evaluate, commit, execute, variants, send)
    ├── internal/ (core logic: template, provider, llm, agent)
    ├── prompt/ (built-in templates)
    ├── llm/ (provider integrations: anthropic, openai)
    └── worker/ (cloud: analytics, webhooks)
  ```
- [x] Core packages explained:
  - [x] `cmd/` - CLI commands using Cobra
  - [x] `internal/template` - YAML template parsing and variable substitution
  - [x] `internal/config` - Config directory management (~/.promptctl/)
  - [x] `llm/` - Provider abstraction (interface-based)
  - [x] `prompt/` - Embedded prompt templates (go:embed)
- [x] Template format:
  - [x] YAML structure (name, description, variables)
  - [x] {{.Variable}} syntax
  - [x] Template lookup (project-local overrides global)
  - [x] Example: code review template
- [x] How a command works:
  - [x] Example: `promptctl review --file=src/auth.ts`
  - [x] Step 1: Load template from ~/.promptctl/templates/review.yaml
  - [x] Step 2: Prompt user for variables (or use CLI flags)
  - [x] Step 3: Read file content
  - [x] Step 4: Render template with variables
  - [x] Step 5: Send to LLM provider
  - [x] Step 6: Stream response to terminal
- [x] Extending promptctl:
  - [x] Add a new command: create file in cmd/, implement interface
  - [x] Add a new provider: implement llm.Provider interface
  - [x] Add templates: drop YAML files in ~/.promptctl/templates/
  - [x] Add project templates: create .promptctl/templates/ in project
- [x] Testing architecture:
  - [x] Unit tests: cmd/*_test.go
  - [x] Mock providers: internal/mock/
  - [x] Integration tests: test templates end-to-end
- [x] Cloud components (for future):
  - [x] worker/: Cloudflare Worker for analytics
  - [x] Separate from core CLI (optional)
- [x] Dependencies:
  - [x] Cobra: CLI framework
  - [x] survey: Interactive prompts
  - [x] YAML: Template parsing
  - [x] No runtime LLM library required (just HTTP)

### 5. Update examples/ directory
Ensure all examples are working and well-documented.

**Tasks:**
- [x] Review examples/ for accuracy
- [x] Test each example (promptctl review, debug, arch, etc.)
- [x] Add example prompt templates if missing
- [x] Document: what each example demonstrates
- [x] Add comments to YAML examples
- [x] Example: review template with security focus
- [x] Example: debug template with error context
- [x] Example: architecture template for decision recording

### 6. Create docs/ROADMAP.md
Public roadmap signals momentum and transparency.

**Content:**
- [ ] Current (v1.0.0): List current features
- [ ] v1.1.0 (planned): Prompt testing framework
  - [ ] Test templates against expected outputs
  - [ ] Regression testing
  - [ ] Model comparison
- [ ] v2.0.0 (vision): Cloud platform
  - [ ] Web dashboard
  - [ ] Prompt registry & versioning
  - [ ] Team collaboration
  - [ ] Analytics
  - [ ] Monetization note: "Cloud features will be paid, CLI is forever free"
- [ ] Future ideas:
  - [ ] IDE extensions (VSCode, JetBrains)
  - [ ] Prompt optimization algorithms
  - [ ] Multi-provider cost optimization
- [ ] How to contribute to roadmap:
  - [ ] Open discussions for ideas
  - [ ] Vote on features
  - [ ] Sponsor development

### 7. Final README review
- [ ] Verify all links work (CONTRIBUTING, INSTALL, ARCHITECTURE, ROADMAP)
- [ ] Check formatting consistency
- [ ] Verify example code is copy-paste ready
- [ ] Test all installation instructions (at least on macOS)
- [ ] Ensure tone is inviting and professional

## Success Criteria
- ✓ README is compelling and enables 5-min onboarding
- ✓ CONTRIBUTING.md makes it easy for developers to add features
- ✓ INSTALL.md covers all platforms clearly
- ✓ ARCHITECTURE.md helps maintainers understand codebase
- ✓ Examples all work and demonstrate real use cases
- ✓ ROADMAP.md signals transparency and growth plans
- ✓ All internal links work
- ✓ No typos or formatting issues

## Notes
- README should be the "front door" - make it great
- Use realistic examples that users can run immediately
- Keep language clear for non-native English speakers
- Emphasize ease of contribution (lower barrier to entry)
- Roadmap transparency builds trust (especially around monetization)
