package cli

import (
	"fmt"
	"maps"
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
// parseArgs applies and drops them wherever they appear before the end-of-flags
// marker; after it they are ordinary text.
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
	"--version": true,
	"-V":        true,
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
	"show": {
		"--field":  {TakesValue: true},
		"--fields": {TakesValue: true},
	},
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

// flagScanLimit caps how many leading arguments ValidateFlags inspects for a
// command whose remaining arguments are free text by definition. A command
// absent from the map is scanned in full.
var flagScanLimit = map[string]int{
	"note add": 1,
}

func init() {
	commandFlags["ready"] = copyFlagsExcept(commandFlags["list"], "--ready", "--blocked")
	commandFlags["blocked"] = copyFlagsExcept(commandFlags["list"], "--blocked", "--ready")
}

// copyFlagsExcept returns a shallow copy of source with the excluded keys removed.
func copyFlagsExcept(source map[string]FlagDef, exclude ...string) map[string]FlagDef {
	result := make(map[string]FlagDef, len(source))
	maps.Copy(result, source)
	for _, e := range exclude {
		delete(result, e)
	}
	return result
}

// ValidateFlags checks that all flag-like arguments in args are valid for the given command.
// A value-taking flag may carry its value attached as "--flag=value" or separated as the
// following argument. Global flags are always accepted, exact-match only. Unknown flags
// produce an error naming the whole argument, with the format:
//
//	unknown flag "{flag}" for "{command}". Run 'tick help {helpCmd}' for usage.
//
// For two-level commands (e.g. "dep add"), the help reference uses the parent command.
// Commands listed in flagScanLimit have only their leading arguments inspected.
func ValidateFlags(command string, args []string, flags CommandFlags) error {
	cmdFlags := flags[command]

	scanEnd := len(args)
	if limit, capped := flagScanLimit[command]; capped {
		scanEnd = min(limit, scanEnd)
	}

	for i := 0; i < scanEnd; i++ {
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

		name, _, attached := cutFlagValue(arg)
		def, ok := cmdFlags[name]
		if !ok || (attached && !def.TakesValue) {
			return fmt.Errorf("unknown flag %q for %q. Run 'tick help %s' for usage.", arg, command, helpCommand(command))
		}

		if def.TakesValue && !attached {
			i++
		}
	}

	return nil
}

// helpCommand returns the command name to use in help references.
// For two-level commands (containing a space), it returns the parent command.
// For single-level commands, it returns the command itself.
func helpCommand(command string) string {
	if before, _, ok := strings.Cut(command, " "); ok {
		return before
	}
	return command
}

// splitLiteralArgs splits the trailing n arguments off args as literals: text the
// caller placed after the end-of-flags marker, which must not be inspected as
// flags. When literals are split off, flagArgs has its capacity clipped, so
// appending to it cannot overwrite the first literal.
func splitLiteralArgs(args []string, n int) (flagArgs, literals []string) {
	n = min(n, len(args))
	if n <= 0 {
		return args, nil
	}
	boundary := len(args) - n
	return args[:boundary:boundary], args[boundary:]
}

// cutFlagValue cuts a flag-shaped argument at its first "=", returning the name
// before it, the value after it, and whether an "=" was present. An argument not
// beginning with "-" is returned whole, with attached false.
func cutFlagValue(arg string) (name, value string, attached bool) {
	if !strings.HasPrefix(arg, "-") {
		return arg, "", false
	}
	name, value, attached = strings.Cut(arg, "=")
	return name, value, attached
}

// flagScanner walks a command's flag arguments, exposing each one whole and cut
// into an attached flag name and value, and resolving a flag's value from either
// spelling. Callers switch on name, read whole for a positional, and call value
// for a flag that takes one.
type flagScanner struct {
	args []string
	i    int
	// whole is the current argument exactly as given.
	whole string
	// name is whole up to its first "=", for a flag-shaped argument.
	name          string
	attachedValue string
	attached      bool
}

func newFlagScanner(args []string) *flagScanner {
	return &flagScanner{args: args, i: -1}
}

// next advances to the following argument, reporting whether one was available.
func (s *flagScanner) next() bool {
	s.i++
	if s.i >= len(s.args) {
		return false
	}
	s.whole = s.args[s.i]
	s.name, s.attachedValue, s.attached = cutFlagValue(s.whole)
	return true
}

// value returns the current flag's value: the attached one when the argument
// carried "=", otherwise the following argument, which it consumes. ok is false
// when a separated value is called for and no argument follows.
func (s *flagScanner) value() (v string, ok bool) {
	if s.attached {
		return s.attachedValue, true
	}
	if s.i+1 >= len(s.args) {
		return "", false
	}
	s.i++
	return s.args[s.i], true
}
