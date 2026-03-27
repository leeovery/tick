package cli

import (
	"fmt"
	"strings"
)

// FlagDef describes a command flag's behavior.
type FlagDef struct {
	// TakesValue indicates whether the flag consumes the next argument as its value.
	TakesValue bool
}

// CommandFlags maps command names to their valid flags and flag definitions.
type CommandFlags map[string]map[string]FlagDef

// globalFlagSet contains all global flags that are accepted by every command.
// These are stripped by parseArgs before dispatch but may appear in subArgs
// when validation runs before global stripping.
var globalFlagSet = map[string]bool{
	"--quiet":   true,
	"-q":        true,
	"--verbose": true,
	"-v":        true,
	"--toon":    true,
	"--pretty":  true,
	"--json":    true,
	"--help":    true,
	"-h":        true,
}

// commandFlags is the central registry of valid per-command flags.
var commandFlags = CommandFlags{
	"init": {},
	"create": {
		"--priority":    {TakesValue: true},
		"--description": {TakesValue: true},
		"--blocked-by":  {TakesValue: true},
		"--blocks":      {TakesValue: true},
		"--parent":      {TakesValue: true},
		"--type":        {TakesValue: true},
		"--tags":        {TakesValue: true},
		"--refs":        {TakesValue: true},
	},
	"update": {
		"--title":             {TakesValue: true},
		"--description":       {TakesValue: true},
		"--priority":          {TakesValue: true},
		"--parent":            {TakesValue: true},
		"--clear-description": {TakesValue: false},
		"--type":              {TakesValue: true},
		"--clear-type":        {TakesValue: false},
		"--tags":              {TakesValue: true},
		"--clear-tags":        {TakesValue: false},
		"--refs":              {TakesValue: true},
		"--clear-refs":        {TakesValue: false},
		"--blocks":            {TakesValue: true},
	},
	"list": {
		"--ready":    {TakesValue: false},
		"--blocked":  {TakesValue: false},
		"--status":   {TakesValue: true},
		"--priority": {TakesValue: true},
		"--parent":   {TakesValue: true},
		"--type":     {TakesValue: true},
		"--tag":      {TakesValue: true},
		"--count":    {TakesValue: true},
	},
	"show":        {},
	"start":       {},
	"done":        {},
	"cancel":      {},
	"reopen":      {},
	"dep add":     {},
	"dep remove":  {},
	"dep tree":    {},
	"note add":    {},
	"note remove": {},
	"remove": {
		"--force": {TakesValue: false},
		"-f":      {TakesValue: false},
	},
	"stats":   {},
	"doctor":  {},
	"rebuild": {},
	"migrate": {
		"--from":         {TakesValue: true},
		"--dry-run":      {TakesValue: false},
		"--pending-only": {TakesValue: false},
	},
}

func init() {
	commandFlags["ready"] = copyFlagsExcept(commandFlags["list"], "--ready", "--blocked")
	commandFlags["blocked"] = copyFlagsExcept(commandFlags["list"], "--blocked", "--ready")
}

// copyFlagsExcept returns a shallow copy of source with the excluded keys removed.
func copyFlagsExcept(source map[string]FlagDef, exclude ...string) map[string]FlagDef {
	result := make(map[string]FlagDef, len(source))
	for k, v := range source {
		result[k] = v
	}
	for _, e := range exclude {
		delete(result, e)
	}
	return result
}

// ValidateFlags checks that all flag-like arguments in args are valid for the given command.
// Global flags are always accepted. Unknown flags produce an error with the format:
//
//	unknown flag "{flag}" for "{command}". Run 'tick help {helpCmd}' for usage.
//
// For two-level commands (e.g. "dep add"), the help reference uses the parent command.
func ValidateFlags(command string, args []string, flags CommandFlags) error {
	cmdFlags := flags[command]

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			continue
		}

		// Numeric values like "-1" are not flags.
		if len(arg) > 1 && arg[1] >= '0' && arg[1] <= '9' {
			continue
		}

		// Global flags are always accepted.
		if globalFlagSet[arg] {
			continue
		}

		def, ok := cmdFlags[arg]
		if !ok {
			return fmt.Errorf("unknown flag %q for %q. Run 'tick help %s' for usage.", arg, command, helpCommand(command))
		}

		// Skip the next argument if this flag takes a value.
		if def.TakesValue {
			i++
		}
	}

	return nil
}

// helpCommand returns the command name to use in help references.
// For two-level commands (containing a space), it returns the parent command.
// For single-level commands, it returns the command itself.
func helpCommand(command string) string {
	if idx := strings.Index(command, " "); idx >= 0 {
		return command[:idx]
	}
	return command
}
