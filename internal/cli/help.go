package cli

import (
	"fmt"
	"io"
)

// flagInfo describes a single command flag for help output.
type flagInfo struct {
	Name     string // "--priority"
	Arg      string // "<0-4>", "<id,...>", "" for bool
	Desc     string // "Task priority (default: 2)"
	Required bool
}

// commandInfo describes a command for help output.
type commandInfo struct {
	Name        string
	Summary     string // one-line for top-level listing
	Usage       string // "tick create <title> [flags]"
	Description string // multi-line detail
	Flags       []flagInfo
}

// commands is the ordered registry of all tick commands.
var commands = []commandInfo{
	{
		Name:    "init",
		Summary: "Initialize a new tick project",
		Usage:   "tick init",
		Description: "Creates a .tick/ directory in the current working directory with an\n" +
			"empty tasks.jsonl file. Errors if already initialized.",
	},
	{
		Name:    "create",
		Summary: "Create a new task",
		Usage:   "tick create <title> [flags]",
		Description: "Creates a new task with the given title. A unique ID is generated\n" +
			"automatically. Priority defaults to 2 (medium).",
		Flags: []flagInfo{
			{"--priority", "<0-4>", "Task priority (default: 2)", false},
			{"--description", "<text>", "Task description", false},
			{"--parent", "<id>", "Parent task ID (creates a subtask)", false},
			{"--blocked-by", "<id,...>", "Task IDs this is blocked by", false},
			{"--blocks", "<id,...>", "Task IDs this blocks", false},
		},
	},
	{
		Name:    "list",
		Summary: "List tasks with optional filters",
		Usage:   "tick list [flags]",
		Description: "Lists tasks with optional filtering by status, priority, parent,\n" +
			"or dependency state. --ready and --blocked are mutually exclusive.",
		Flags: []flagInfo{
			{"--status", "<open|in_progress|done|cancelled>", "Filter by status", false},
			{"--priority", "<0-4>", "Filter by priority", false},
			{"--parent", "<id>", "Filter by parent task", false},
			{"--ready", "", "Show only ready tasks (no unresolved blockers)", false},
			{"--blocked", "", "Show only blocked tasks", false},
		},
	},
	{
		Name:        "show",
		Summary:     "Show full task detail",
		Usage:       "tick show <task-id>",
		Description: "Displays complete details for a single task including description,\ndependencies, subtasks, and timestamps.",
	},
	{
		Name:    "update",
		Summary: "Update task fields",
		Usage:   "tick update <task-id> [flags]",
		Description: "Updates one or more fields on an existing task.\n" +
			"At least one flag is required.",
		Flags: []flagInfo{
			{"--title", "<text>", "New title", false},
			{"--description", "<text>", "New description", false},
			{"--clear-description", "", "Remove description", false},
			{"--priority", "<0-4>", "New priority", false},
			{"--parent", "<id>", "New parent task ID", false},
			{"--blocks", "<id,...>", "Task IDs this blocks", false},
		},
	},
	{
		Name:        "start",
		Summary:     "Start a task (open → in_progress)",
		Usage:       "tick start <task-id>",
		Description: "Transitions a task from open to in_progress.",
	},
	{
		Name:        "done",
		Summary:     "Complete a task (in_progress → done)",
		Usage:       "tick done <task-id>",
		Description: "Transitions a task from in_progress to done.",
	},
	{
		Name:        "cancel",
		Summary:     "Cancel a task (any → cancelled)",
		Usage:       "tick cancel <task-id>",
		Description: "Cancels a task regardless of current status.",
	},
	{
		Name:        "reopen",
		Summary:     "Reopen a task (done/cancelled → open)",
		Usage:       "tick reopen <task-id>",
		Description: "Transitions a done or cancelled task back to open.",
	},
	{
		Name:    "dep",
		Summary: "Manage task dependencies",
		Usage:   "tick dep <add|rm> <task-id> <blocked-by-id>",
		Description: "Adds or removes a dependency between two tasks.\n" +
			"Prevents self-references and dependency cycles.",
	},
	{
		Name:    "ready",
		Summary: "List ready tasks (alias: list --ready)",
		Usage:   "tick ready [flags]",
		Description: "Lists tasks that have no unresolved blockers. Alias for list --ready.\n" +
			"Accepts the same additional filters as list.",
		Flags: []flagInfo{
			{"--status", "<open|in_progress|done|cancelled>", "Filter by status", false},
			{"--priority", "<0-4>", "Filter by priority", false},
			{"--parent", "<id>", "Filter by parent task", false},
		},
	},
	{
		Name:    "blocked",
		Summary: "List blocked tasks (alias: list --blocked)",
		Usage:   "tick blocked [flags]",
		Description: "Lists tasks that have unresolved blockers. Alias for list --blocked.\n" +
			"Accepts the same additional filters as list.",
		Flags: []flagInfo{
			{"--status", "<open|in_progress|done|cancelled>", "Filter by status", false},
			{"--priority", "<0-4>", "Filter by priority", false},
			{"--parent", "<id>", "Filter by parent task", false},
		},
	},
	{
		Name:        "stats",
		Summary:     "Show task statistics",
		Usage:       "tick stats",
		Description: "Displays summary statistics: task counts by status and priority.",
	},
	{
		Name:        "rebuild",
		Summary:     "Rebuild SQLite cache from JSONL",
		Usage:       "tick rebuild",
		Description: "Forces a full rebuild of the SQLite cache from tasks.jsonl.",
	},
	{
		Name:        "doctor",
		Summary:     "Run diagnostic checks",
		Usage:       "tick doctor",
		Description: "Runs diagnostic checks on the tick data: JSONL syntax, ID format,\nduplicates, orphaned references, dependency cycles, and cache staleness.",
	},
	{
		Name:    "migrate",
		Summary: "Import tasks from external tools",
		Usage:   "tick migrate --from <provider> [flags]",
		Description: "Imports tasks from an external tool into tick.\n" +
			"Currently supported providers: beads.",
		Flags: []flagInfo{
			{"--from", "<provider>", "Source provider (required)", true},
			{"--dry-run", "", "Preview without importing", false},
			{"--pending-only", "", "Import only pending/open tasks", false},
		},
	},
	{
		Name:        "help",
		Summary:     "Show help for a command",
		Usage:       "tick help [<command>]",
		Description: "Shows usage information. With no argument, lists all commands.\nWith a command name, shows detailed help for that command.",
	},
}

// findCommand returns the commandInfo for the given name, or nil.
func findCommand(name string) *commandInfo {
	for i := range commands {
		if commands[i].Name == name {
			return &commands[i]
		}
	}
	return nil
}

// printTopLevelHelp writes the full command listing to w.
func printTopLevelHelp(w io.Writer) {
	fmt.Fprintln(w, "Usage: tick <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, cmd := range commands {
		fmt.Fprintf(w, "  %-14s%s\n", cmd.Name, cmd.Summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Global flags:")
	fmt.Fprintln(w, "  --quiet, -q     Suppress output")
	fmt.Fprintln(w, "  --verbose, -v   Show debug information")
	fmt.Fprintln(w, "  --toon          Force TOON output format")
	fmt.Fprintln(w, "  --pretty        Force pretty output format")
	fmt.Fprintln(w, "  --json          Force JSON output format")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'tick help <command>' for detailed help on a command.")
	fmt.Fprintln(w, "Run 'tick help --all' for complete reference of all commands and flags.")
}

// printAllHelp writes compact, concatenated help for every command to w.
// Designed for AI agents to discover the full CLI surface in one call.
func printAllHelp(w io.Writer) {
	fmt.Fprintln(w, "Global flags: --help/-h --quiet/-q --verbose/-v --toon --pretty --json")
	fmt.Fprintln(w)
	for i, cmd := range commands {
		fmt.Fprintln(w, cmd.Usage)
		fmt.Fprintf(w, "  %s\n", cmd.Summary)
		for _, f := range cmd.Flags {
			label := f.Name
			if f.Arg != "" {
				label += " " + f.Arg
			}
			fmt.Fprintf(w, "  %-24s%s\n", label, f.Desc)
		}
		if i < len(commands)-1 {
			fmt.Fprintln(w)
		}
	}
}

// printCommandHelp writes detailed help for a single command to w.
func printCommandHelp(w io.Writer, cmd *commandInfo) {
	fmt.Fprintf(w, "Usage: %s\n", cmd.Usage)
	fmt.Fprintln(w)
	fmt.Fprintln(w, cmd.Description)

	if len(cmd.Flags) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Flags:")
		for _, f := range cmd.Flags {
			label := f.Name
			if f.Arg != "" {
				label += " " + f.Arg
			}
			fmt.Fprintf(w, "  %-24s%s\n", label, f.Desc)
		}
	}
}
