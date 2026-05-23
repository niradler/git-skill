package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/niradler/git-skill/internal/gitops"
	"github.com/niradler/git-skill/internal/state"
)

// stringSliceFlag implements flag.Value for repeatable string flags.
type stringSliceFlag []string

func (s *stringSliceFlag) String() string     { return strings.Join(*s, ",") }
func (s *stringSliceFlag) Set(v string) error { *s = append(*s, v); return nil }

func Add(p Profile, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	remote := fs.String("from", "", "remote URL (required)")
	runtimesFlag := fs.String("runtime", "", "comma-separated runtime names; omitted = canonical only")
	dev := fs.Bool("dev", false, "install all files (symlink/junction); default is prod (copy + .assetignore)")
	kindFlag := fs.String("kind", "", "asset kind (skill|agent); inferred from profile when omitted")
	var targets stringSliceFlag
	fs.Var(&targets, "target", "override install path for a runtime, form <runtime>=<path>; repeatable")

	// Reorder args so flags come before positional arguments, because flag.Parse
	// stops at the first non-flag argument. Users may write: add <name> --from <url>.
	flags, positional := splitFlagsAndPositional(args, boolFlagNames(fs))
	flagArgs := append(flags, positional...)
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("add <name>[@<version-spec>] --from <url>")
	}
	if *remote == "" {
		return errors.New("--from <url> is required")
	}

	name, spec := splitNameSpec(fs.Arg(0))
	k, err := pickKind(p, *kindFlag)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	st, err := state.Read(cwd)
	if err != nil {
		return err
	}

	remoteAssets, err := gitops.ListRemote(*remote)
	if err != nil {
		return fmt.Errorf("ls-remote %s: %w", *remote, err)
	}
	resolved, err := ResolveSpec(remoteAssets, k, name, spec)
	if err != nil {
		return err
	}

	rtMap, err := buildRuntimes(*runtimesFlag, targets)
	if err != nil {
		return err
	}
	canonical := filepath.Join(st.Root(k), name)
	specForState := spec
	if specForState == "" {
		if resolved.Version != "" {
			specForState = "^" + resolved.Version
		} else {
			specForState = "latest"
		}
	}

	entry := state.Entry{
		Spec:      specForState,
		Remote:    *remote,
		Runtimes:  rtMap,
		Dev:       *dev,
		Version:   resolved.Version,
		Commit:    resolved.Commit,
		Canonical: canonical,
	}
	st.Set(k, name, entry)
	if err := st.Write(cwd); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "+ %s %s @ %s\n", k, name, resolved.Version)
	return installOne(cwd, st, k, name, entry, stdout)
}

// buildRuntimes merges the --runtime list and the --target overrides into
// a single runtime override map. Every --target name must also appear in
// --runtime (or the map seeded from it).
func buildRuntimes(runtimeList string, targets []string) (map[string]state.RuntimeOverride, error) {
	out := map[string]state.RuntimeOverride{}
	for _, p := range strings.Split(runtimeList, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out[p] = state.RuntimeOverride{}
	}
	for _, t := range targets {
		i := strings.Index(t, "=")
		if i <= 0 || i == len(t)-1 {
			return nil, fmt.Errorf("--target %q: expected <runtime>=<path>", t)
		}
		rt, path := t[:i], t[i+1:]
		cur, ok := out[rt]
		if !ok {
			return nil, fmt.Errorf("--target %s: runtime not in --runtime list", rt)
		}
		cur.To = path
		out[rt] = cur
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// splitFlagsAndPositional separates flag args (starting with "-") from
// positional args so that flags and positionals can appear in any order.
// It handles both "-flag value" and "-flag=value" forms. Bool flags listed
// in boolFlags do not consume the following arg as their value.
func splitFlagsAndPositional(args []string, boolFlags map[string]bool) (flags, positional []string) {
	i := 0
	for i < len(args) {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			// Check if it's a flag=value form.
			if strings.Contains(a, "=") {
				flags = append(flags, a)
				i++
				continue
			}
			flags = append(flags, a)
			i++
			// Bool flags never take a value argument.
			if boolFlags[flagName(a)] {
				continue
			}
			// If next arg doesn't start with "-", it's the flag's value.
			if i < len(args) && !strings.HasPrefix(args[i], "-") {
				flags = append(flags, args[i])
				i++
			}
		} else {
			positional = append(positional, a)
			i++
		}
	}
	return flags, positional
}

// flagName strips leading dashes from a flag token: "--dev" → "dev".
func flagName(s string) string {
	return strings.TrimLeft(s, "-")
}

// boolFlagNames extracts the names of bool flags from a FlagSet so
// splitFlagsAndPositional can leave them alone when reordering.
func boolFlagNames(fs *flag.FlagSet) map[string]bool {
	out := map[string]bool{}
	fs.VisitAll(func(f *flag.Flag) {
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
			out[f.Name] = true
		}
	})
	return out
}

func splitNameSpec(s string) (name, spec string) {
	if i := strings.Index(s, "@"); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}
