# Promptctl Phase 1: Open-Source Readiness

You are automating the open-source launch prep for promptctl, a Go CLI tool for prompt engineering templates.

## Context
- Repo: /Users/olegkoval/projects/personal/active/prompt-ctl.com/promptctl
- Goal: Prepare codebase for public GitHub release
- Current state: v1.0.0 ready, needs cleanup and documentation

## Tasks

### 1. Audit & Remove Secrets
- [x] Scan cmd/, internal/, config/ for hardcoded API keys, tokens, secrets
- [x] Check CASE_FOR_PROMPTCTL.md for sensitive info
- [x] Verify no .env files in git history
- [x] Scan .promptctl/ directory for credentials
- [x] Create .gitignore entries for sensitive files
- Report findings and remove any found secrets

### 2. Code Cleanup
- [x] Review cmd/ directory - identify unfinished/experimental commands
- [x] Check internal/ for TODO comments, hacks, or incomplete features
- [x] Remove or mark as experimental any beta/alpha code
- [x] Delete temporary files: .DS_Store, coverage.out, .claude/, .cursor/
- [x] Verify dist/ is properly .gitignored (rebuilt on release)
- [x] Ensure node_modules is .gitignored

### 3. Generate Documentation
Create the following markdown files:

**README.md** (update existing)
- One-sentence description
- Problem statement (why promptctl exists)
- Installation section (apt, yum, brew, go install, direct binary)
- Quick start example (5 minutes)
- Feature showcase
- Comparison with alternatives (explain unique value)
- Roadmap section
- Contributing section with link to CONTRIBUTING.md

**docs/CONTRIBUTING.md** (new)
- Developer setup (go version, cloning, dependencies)
- Running tests: go test ./...
- Code style: go fmt, go vet
- PR process and expectations
- Commit message guidelines
- Testing requirements

**docs/INSTALL.md** (new)
- Linux: apt, yum, pacman, snap, direct binary
- macOS: brew, direct binary, go install
- Windows: direct binary, go install
- Platform-specific instructions
- Verification: how to check installation worked

**docs/ARCHITECTURE.md** (new)
- Code structure overview (cmd/, internal/, llm/, prompt/, worker/)
- How prompt templates work (YAML format, variables, placeholders)
- How to extend: adding new commands
- Provider adapters explanation
- Testing framework

### 4. Code Quality Verification
- [x] Run `go vet ./...` - no errors
- [x] Run `go fmt ./...` - apply formatting
- [x] Run `go test ./...` - all tests pass
- [x] Check test coverage
- [x] Ensure CI/CD (GitHub Actions) configuration exists and works

### 5. Legal & License
- [ ] Create LICENSE file with Apache 2.0 license text
- [ ] Create AUTHORS.md with copyright holder
- [ ] Review CHANGELOG.md for accuracy
- [ ] Verify version in cmd/root.go matches v1.0.0

### 6. Final Verification
- [ ] All files are readable (no permission issues)
- [ ] No build artifacts in tracked files
- [ ] .gitignore is comprehensive
- [ ] Examples/ directory has working examples
- [ ] docs/ directory has all required documentation

## Success Criteria
- ✓ No secrets found in codebase
- ✓ All experimental code removed or documented
- ✓ All required docs generated with quality content
- ✓ go vet, go fmt, go test all pass
- ✓ Apache 2.0 license in place
- ✓ Fresh clone can build and run successfully

## Notes
- Keep CASE_FOR_PROMPTCTL.md - it's good marketing material
- Focus on clarity in docs - assume reader is new to promptctl
- Examples should be copy-paste ready
- Roadmap should mention potential cloud/paid tier (open core strategy)
