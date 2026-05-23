package gitops

import (
	"sort"
	"strings"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/kind"
)

type RemoteAsset struct {
	Kind     kind.Kind
	Name     string
	Commit   string            // SHA at refs/assets/<kind>/<name> tip
	Versions []string          // sorted ascending (tag suffix, e.g. "v1.0.0")
	Commits  map[string]string // version → SHA for refs/asset-tags/<kind>/<name>/<version>
}

// CommitForVersion returns the commit SHA pinned by the given version tag.
// Returns empty string if the tag wasn't observed on the remote.
func (a RemoteAsset) CommitForVersion(v string) string {
	if a.Commits == nil {
		return ""
	}
	return a.Commits[v]
}

func ListRemote(url string) ([]RemoteAsset, error) {
	out, err := git.RunLines("ls-remote", url)
	if err != nil {
		return nil, err
	}
	return parseLsRemote(out)
}

func parseLsRemote(lines []string) ([]RemoteAsset, error) {
	byName := map[string]*RemoteAsset{}
	for _, line := range lines {
		tab := strings.IndexByte(line, '\t')
		if tab < 0 {
			continue
		}
		sha := line[:tab]
		ref := line[tab+1:]

		switch {
		case strings.HasPrefix(ref, "refs/assets/"):
			rest := strings.TrimPrefix(ref, "refs/assets/")
			kindStr, name, ok := strings.Cut(rest, "/")
			if !ok || name == "" {
				continue
			}
			k, err := kind.Parse(kindStr)
			if err != nil {
				continue
			}
			key := kindStr + "/" + name
			a := byName[key]
			if a == nil {
				a = &RemoteAsset{Kind: k, Name: name}
				byName[key] = a
			}
			a.Commit = sha

		case strings.HasPrefix(ref, "refs/asset-tags/"):
			rest := strings.TrimPrefix(ref, "refs/asset-tags/")
			kindStr, rest2, ok := strings.Cut(rest, "/")
			if !ok {
				continue
			}
			lastSlash := strings.LastIndex(rest2, "/")
			if lastSlash < 0 {
				continue
			}
			name := rest2[:lastSlash]
			version := rest2[lastSlash+1:]
			k, err := kind.Parse(kindStr)
			if err != nil {
				continue
			}
			key := kindStr + "/" + name
			a := byName[key]
			if a == nil {
				a = &RemoteAsset{Kind: k, Name: name}
				byName[key] = a
			}
			a.Versions = append(a.Versions, version)
			if a.Commits == nil {
				a.Commits = make(map[string]string)
			}
			a.Commits[version] = sha
		}
	}
	out := make([]RemoteAsset, 0, len(byName))
	for _, a := range byName {
		sort.Strings(a.Versions)
		out = append(out, *a)
	}
	return out, nil
}
