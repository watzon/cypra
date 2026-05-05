#!/usr/bin/env bash
#
# bitforge installer - wires bundled command prompts into OpenCode slash
# commands under .opencode/commands/bitforge/ and source-command skills under
# .agents/skills/source-command-bitforge-*.
#
# Idempotent. Safe to re-run after pulling updates to the skill bundle.
#
# Usage (from a workspace root):
#   bash .agents/skills/bitforge/install.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_NAME="bitforge"

SKILLS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
AGENTS_DIR="$(cd "$SKILLS_DIR/.." && pwd)"
WORKSPACE_DIR="$(cd "$AGENTS_DIR/.." && pwd)"
COMMANDS_SOURCE="$SCRIPT_DIR/commands"
SLASH_COMMANDS_TARGET="$WORKSPACE_DIR/.opencode/commands/$SKILL_NAME"

if [[ "$(basename "$SKILLS_DIR")" != "skills" ]]; then
  echo "error: bitforge bundle is not at <workspace>/.agents/skills/bitforge" >&2
  echo "       this script lives at: $SCRIPT_DIR" >&2
  echo "       resolved skills dir: $SKILLS_DIR" >&2
  exit 1
fi

if [[ "$(basename "$AGENTS_DIR")" != ".agents" ]]; then
  echo "error: bitforge bundle is not under a .agents directory" >&2
  echo "       resolved agents dir: $AGENTS_DIR" >&2
  exit 1
fi

if [[ ! -d "$COMMANDS_SOURCE" ]]; then
  echo "error: command sources not found at $COMMANDS_SOURCE" >&2
  exit 1
fi

description_for() {
  case "$1" in
    brainstorm) echo "Start the bitforge brainstorm phase - produce or refine BRAINSTORM.md" ;;
    plan) echo "Produce or refine PLAN.md (PRD + SDD) from BRAINSTORM.md" ;;
    design) echo "Produce or refine DESIGN.md (visual design specification) from PLAN.md" ;;
    pencil) echo "Emit a Pencil handoff prompt the user can paste into a Pencil session to wireframe the project" ;;
    taskify) echo "Produce TASKS.md from PLAN.md (and DESIGN.md) using the bitforge template" ;;
    next) echo "Pick up and execute the next phase from TASKS.md" ;;
    status) echo "Show the current bitforge pipeline state - which artifacts exist, where the project is, what to do next" ;;
    close-phase) echo "Verify acceptance, fill the Handoff, and mark the current phase complete" ;;
    *) echo "Run the bitforge $1 command" ;;
  esac
}

skills_installed=0
skills_updated=0
commands_installed=0
commands_updated=0

mkdir -p "$SLASH_COMMANDS_TARGET"

for cmd in "$COMMANDS_SOURCE"/*.md; do
  [[ -e "$cmd" ]] || continue

  command_name="$(basename "$cmd" .md)"
  skill_name="source-command-$SKILL_NAME-$command_name"
  target_dir="$SKILLS_DIR/$skill_name"
  target_file="$target_dir/SKILL.md"
  description="$(description_for "$command_name")"

  mkdir -p "$target_dir"

  if [[ -e "$target_file" ]]; then
    skills_updated=$((skills_updated + 1))
  else
    skills_installed=$((skills_installed + 1))
  fi

  {
    printf '%s\n' '---'
    printf 'name: "%s"\n' "$skill_name"
    printf 'description: "%s"\n' "$description"
    printf '%s\n\n' '---'
    printf '# %s\n\n' "$skill_name"
    printf 'Use this skill when the user asks to run `$%s` or the legacy `/%s:%s` command.\n\n' "$skill_name" "$SKILL_NAME" "$command_name"
    printf '## Command Template\n\n'
    awk 'BEGIN { frontmatter=0; done=0 } /^---$/ && done == 0 { frontmatter++; if (frontmatter == 2) { done=1; next } next } done == 1 { print }' "$cmd"
  } > "$target_file"

  slash_target="$SLASH_COMMANDS_TARGET/$command_name.md"
  if [[ -e "$slash_target" ]]; then
    commands_updated=$((commands_updated + 1))
  else
    commands_installed=$((commands_installed + 1))
  fi
  cp "$cmd" "$slash_target"
done

echo "bitforge OpenCode commands installed."
echo "  slash commands installed: $commands_installed"
echo "  slash commands updated:   $commands_updated"
echo "  source skills installed:  $skills_installed"
echo "  source skills updated:    $skills_updated"
echo
echo "Slash commands available in this workspace after restarting OpenCode:"
echo "  /bitforge:brainstorm"
echo "  /bitforge:plan"
echo "  /bitforge:design"
echo "  /bitforge:pencil"
echo "  /bitforge:taskify"
echo "  /bitforge:next"
echo "  /bitforge:status"
echo "  /bitforge:close-phase"
