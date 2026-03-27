import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { execSync } from "child_process";
import * as path from "path";
import * as fs from "fs";

export default function (pi: ExtensionAPI) {
  /**
   * /promptctl command: Render a promptctl template and pipe to current model
   * Usage: /promptctl review --file=src/auth.ts
   *        /promptctl debug --file=src/service.ts --error="ECONNREFUSED"
   */
  pi.registerCommand("promptctl", {
    description: "Render a promptctl template and pipe to current model",
    handler: async ({ args, sendMessage }) => {
      try {
        const [templateName, ...cmdArgs] = args.split(/\s+/);

        if (!templateName) {
          sendMessage(
            "Usage: /promptctl <template-name> [--var=value ...]\n\nRun /quick-templates to see available templates.",
            { type: "info" }
          );
          return;
        }

        const renderedPrompt = execSync(
          `promptctl run ${templateName} ${cmdArgs.join(" ")}`,
          { encoding: "utf-8" }
        );

        sendMessage(renderedPrompt, { type: "user" });
      } catch (err: any) {
        sendMessage(
          `promptctl error: ${err.message}\n\nRun /quick-templates to see available templates.`,
          { type: "error" }
        );
      }
    },
  });

  /**
   * /quick-templates: List available promptctl templates
   */
  pi.registerCommand("quick-templates", {
    description: "List available promptctl templates for your workflow",
    handler: async ({ sendMessage }) => {
      try {
        const output = execSync("promptctl list", { encoding: "utf-8" });
        sendMessage(`## Available Templates\n\n${output}`, { type: "info" });
      } catch (err: any) {
        sendMessage(
          `Templates unavailable: ${err.message}\n\nMake sure promptctl is installed: https://github.com/oleg-koval/promptctl`,
          { type: "error" }
        );
      }
    },
  });

  /**
   * /cost-score: Score a prompt file for efficiency (0-100)
   * Wraps promptctl score to give quick feedback on prompt quality
   */
  pi.registerCommand("cost-score", {
    description: "Score a prompt file for efficiency (0-100)",
    handler: async ({ args, sendMessage }) => {
      try {
        const filePath = args.trim();

        if (!filePath) {
          sendMessage(
            "Usage: /cost-score <path-to-prompt-file>\n\nScores a prompt on structure, clarity, constraints, and persona.",
            { type: "info" }
          );
          return;
        }

        const scoreOutput = execSync(
          `promptctl score ${filePath} --format=json`,
          { encoding: "utf-8" }
        ).trim();

        const result = JSON.parse(scoreOutput);
        const file = result.files && result.files[0];

        if (!file) {
          sendMessage("No score results returned.", { type: "warning" });
          return;
        }

        const status =
          file.score >= 80
            ? "Good prompt structure"
            : file.score >= 60
              ? "Could be tighter"
              : "Needs optimization";

        sendMessage(
          `Prompt Efficiency Score: ${file.score}/100 — ${status}\n\nTip: Use \`promptctl fix ${filePath}\` to auto-improve low-scoring prompts.`,
          { type: "info" }
        );
      } catch (err: any) {
        sendMessage(
          `Score unavailable: ${err.message}`,
          { type: "warning" }
        );
      }
    },
  });

  /**
   * promptctl_apply tool: Let the LLM call promptctl directly
   * The LLM can invoke this to render a template and inject the result
   */
  pi.registerTool("promptctl_apply", {
    description:
      "Render a promptctl template with variables and return the structured prompt. Use this when the user needs a high-quality structured prompt for a code task.",
    parameters: {
      type: "object",
      properties: {
        template: {
          type: "string",
          description:
            "Template name (e.g. review, debug, arch, commit, explain). Run `promptctl list` to see all available templates.",
        },
        vars: {
          type: "object",
          description:
            "Key-value pairs for template variables (e.g. { file: 'src/auth.ts', focus: 'security' })",
          additionalProperties: { type: "string" },
        },
      },
      required: ["template"],
    },
    handler: async ({ template, vars = {} }: { template: string; vars?: Record<string, string> }) => {
      const varArgs = Object.entries(vars)
        .map(([k, v]) => `--${k}="${v}"`)
        .join(" ");

      try {
        const output = execSync(`promptctl run ${template} ${varArgs}`, {
          encoding: "utf-8",
        });
        return { result: output };
      } catch (err: any) {
        return {
          error: `promptctl failed: ${err.message}. Run \`promptctl list\` to see available templates.`,
        };
      }
    },
  });

  pi.on("extension_loaded", ({}, ctx) => {
    ctx.notify(
      "promptctl integration loaded. Try: /quick-templates or /promptctl review --file=<path>"
    );
  });
}
