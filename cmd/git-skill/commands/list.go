package commands

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/refs"
	"github.com/niradler/git-skill/internal/semver"
)

func List(p Profile, args []string, stdout, stderr io.Writer) error {
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "KIND\tNAME\tCOMMIT\tLATEST_TAG")

	for _, k := range profileKinds(p) {
		assets, err := listAssetsByKind(k)
		if err != nil {
			return err
		}
		for _, a := range assets {
			short := a.commit
			if len(short) > 7 {
				short = short[:7]
			}
			latest, err := latestTag(k, a.name)
			if err != nil {
				return err
			}
			if latest == "" {
				latest = "(none)"
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", k, a.name, short, latest)
		}
	}
	return tw.Flush()
}

type assetEntry struct{ name, commit string }

func listAssetsByKind(k kind.Kind) ([]assetEntry, error) {
	lines, err := git.RunLines("for-each-ref", "--format=%(refname) %(objectname)", refs.KindPrefix(k))
	if err != nil {
		return nil, err
	}
	var out []assetEntry
	for _, ln := range lines {
		ref, commit, ok := strings.Cut(ln, " ")
		if !ok {
			continue
		}
		_, name, err := refs.ParseRef(ref)
		if err != nil {
			continue
		}
		out = append(out, assetEntry{name: name, commit: commit})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

func latestTag(k kind.Kind, name string) (string, error) {
	prefix := refs.KindTagPrefix(k) + name + "/"
	lines, err := git.RunLines("for-each-ref", "--format=%(refname)", prefix)
	if err != nil {
		return "", err
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
	if len(versions) == 0 {
		return "", nil
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i].Compare(versions[j]) > 0 })
	return versions[0].String(), nil
}
