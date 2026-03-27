# Roadmap

This roadmap outlines where promptctl is headed. It reflects our current plans, but priorities may shift based on community feedback and real-world usage.

> **Open core model:** The CLI is forever free and open source (Apache 2.0). Cloud features will be a paid product that funds continued development of the free CLI.

## Current Release: v1.0.0

The foundation is shipped. promptctl is a fully functional CLI for prompt engineering workflows:

- **Template engine** - YAML-based prompt templates with `{{.variable}}` substitution
- **Starter templates** - Built-in templates for review, debug, arch, commit, and explain workflows
- **Project-level overrides** - `.promptctl/templates/` in your repo overrides global templates
- **LLM integration** - Send prompts directly to Anthropic (Claude) or OpenAI (GPT) from the terminal
- **Cost estimation** - Estimate token costs before sending, compare across models
- **Prompt scoring** - Score prompt files 0-100 on structure, clarity, constraints, and persona
- **Auto-fix** - Automatically improve low-scoring prompts
- **Experimentation** - Benchmark templates across models, auto-generate and rank variants
- **AI-powered creation** - Transform raw intent into structured prompts with `promptctl create`
- **CI/CD friendly** - Exit codes, JSON output, and `--min-score` gates for pipelines
- **Clipboard integration** - Copy rendered prompts to clipboard with `promptctl cp`

## Planned: v1.1.0 - Prompt Testing Framework

Treat prompts like code means testing them like code. This release adds a testing layer:

- **Test definitions** - Define expected behaviors for templates (e.g., "review template should mention security for auth files")
- **Assertion types** - Check for keyword presence, output structure, tone, and length constraints
- **Regression testing** - Detect when prompt changes degrade output quality
- **Model comparison** - Run the same template across models and compare results side-by-side
- **Test reports** - Machine-readable test results for CI integration
- **Snapshot testing** - Save baseline outputs and diff against future runs

## Vision: v2.0.0 - Cloud Platform

A web layer for teams that need collaboration, analytics, and governance:

- **Web dashboard** - Browse, edit, and manage prompt templates in the browser
- **Prompt registry** - Publish and version templates for your team or the community
- **Team collaboration** - Shared template libraries with role-based access control
- **Analytics** - Track prompt usage, costs, quality scores, and trends over time
- **A/B testing** - Run controlled experiments on prompt variants with real traffic
- **Audit trail** - Full history of prompt changes and who made them

> Cloud features will be paid. The CLI remains free. Revenue from the cloud platform funds development of the open-source CLI.

## Future Ideas

These are ideas we're excited about but haven't committed to a timeline:

- **IDE extensions** - VSCode and JetBrains plugins for template editing with autocomplete and preview
- **Prompt optimization** - Algorithmic prompt compression and restructuring for cost reduction
- **Multi-provider cost optimization** - Automatically route prompts to the cheapest provider that meets quality thresholds
- **Template marketplace** - Community-contributed templates for common workflows (code review, documentation, testing)
- **Language server** - LSP support for YAML template files with validation and hover docs
- **Git hooks** - Pre-commit hooks that score prompts and block low-quality commits

## Contributing to the Roadmap

We build in the open and prioritize based on community input.

- **Propose ideas** - Open a [GitHub Discussion](https://github.com/oleg-koval/promptctl/discussions) to suggest features or share use cases
- **Vote on features** - React to existing discussions to signal interest. The most-requested features get prioritized
- **Contribute code** - See [CONTRIBUTING.md](CONTRIBUTING.md) for how to submit PRs. Feature PRs that align with the roadmap are welcome
- **Sponsor development** - Sponsoring helps us dedicate more time to open-source work. See our [GitHub Sponsors](https://github.com/sponsors/oleg-koval) page

## Status Key

| Status | Meaning |
|--------|---------|
| Shipped | Available in the current release |
| Planned | Committed to a future release |
| Exploring | Under consideration, not yet committed |
| Community | Driven by community contributions |
