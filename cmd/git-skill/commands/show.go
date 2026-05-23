package commands

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/refs"
	"github.com/niradler/git-skill/internal/semver"
)

func Show(p Profile, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	kindFlag := fs.String("kind", "", "skill|agent (inferred when omitted)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("show <name>")
	}
	name := fs.Arg(0)
	k, err := resolveAssetKind(name, *kindFlag)
	if err != nil {
		return err
	}

	commit, err := git.ResolveRef(refs.Ref(k, name))
	if err != nil {
		return err
	}
	short := commit
	if len(short) > 7 {
		short = short[:7]
	}
	fmt.Fprintf(stdout, "%s %s\ncommit: %s\n", k, name, short)

	// Tag list, sorted high -> low
	tagPrefix := refs.KindTagPrefix(k) + name + "/"
	lines, err := git.RunLines("for-each-ref", "--format=%(refname)", tagPrefix)
	if err != nil {
		return err
	}
	var versions []semver.Version
	for _, ref := range lines {
		_, _, ver, err := refs.ParseTagRef(ref)
		if err != nil {
			continue
		}
		v, err := semver.Parse(ver)
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i].Compare(versions[j]) > 0 })

	if len(versions) == 0 {
		fmt.Fprintln(stdout, "tags: (none)")
	} else {
		var ss []string
		for _, v := range versions {
			ss = append(ss, v.String())
		}
		fmt.Fprintf(stdout, "tags: %s\n", strings.Join(ss, ", "))
	}
	return nil
}
