# Example Templates

This directory contains example prompt templates for promptctl. Each template
demonstrates a different use case and can be used as-is or customized for your
projects.

## Quick Start

Copy any template to your project or global templates directory:

```bash
# Project-local (takes precedence)
mkdir -p .promptctl/templates
cp examples/review-security.yaml .promptctl/templates/review-security.yaml

# Global (available everywhere)
cp examples/review-security.yaml ~/.promptctl/templates/review-security.yaml
```

Then run it:

```bash
promptctl run review-security --file=src/auth.ts
```

## Templates

| File | Use Case | Key Variables |
|------|----------|---------------|
| `review-security.yaml` | Security-focused code review | `file` (required), `focus` |
| `debug-error-context.yaml` | Debug errors with full context | `file` (required), `error_message` (required), `stack_trace` |
| `architecture-decision.yaml` | Record architecture decisions (ADR) | `title` (required), `context` (required), `options` |
| `commit-changelog.yaml` | Generate commit messages from diffs | `diff` (required), `ticket` |
| `api-review.yaml` | Review API endpoint design | `file` (required), `api_style` |

## Template Format

All templates use the same YAML structure:

```yaml
name: template-name
description: What this template does

variables:
  - name: file
    description: Path to the input file
    required: true
  - name: focus
    description: Optional focus area
    default: general

body: |
  Your prompt text here.
  Use {{.variable_name}} for substitution.
  Use {{if .variable}}conditional text{{end}} for optional sections.
```

### Auto-Variables

When you pass `--file=path/to/file`, these variables are automatically available:

- `{{.file_content}}` - The full contents of the file
- `{{.file_name}}` - The filename (basename only)
- `{{.file_ext}}` - The file extension (without the dot)

## Creating Your Own Templates

```bash
# Scaffold a new template
promptctl create my-template

# Or manually create a YAML file
cat > ~/.promptctl/templates/my-template.yaml << 'EOF'
name: my-template
description: My custom template

variables:
  - name: file
    description: Path to analyze
    required: true

body: |
  Analyze this code:

  <file name="{{.file_name}}">
  {{.file_content}}
  </file>

  Provide your analysis.
EOF
```

## Tips

- Local templates (`.promptctl/templates/`) override global ones with the same name
- Use `promptctl list` to see all available templates
- Use `promptctl show <name>` to view a template's content
- Use `promptctl score <name>` to check template quality
- Templates support `{{if .var}}...{{end}}` conditionals for optional sections
