package commands

import (
	"fmt"

	"github.com/niradler/git-skill/internal/gitops"
	"github.com/niradler/git-skill/internal/kind"
	"github.com/niradler/git-skill/internal/semver"
)

type Resolved struct {
	Commit  string
	Version string
}

func ResolveSpec(remote []gitops.RemoteAsset, k kind.Kind, name, spec string) (Resolved, error) {
	for _, ra := range remote {
		if ra.Kind != k || ra.Name != name {
			continue
		}
		if spec == "" {
			spec = "latest"
		}
		sp, err := semver.ParseSpec(spec)
		if err != nil {
			return Resolved{}, err
		}
		if sp.Op == semver.OpLatest && len(ra.Versions) == 0 {
			return Resolved{Commit: ra.Commit}, nil
		}
		parsed := make([]semver.Version, 0, len(ra.Versions))
		versionByParsed := map[string]string{}
		for _, raw := range ra.Versions {
			v, err := semver.Parse(raw)
			if err != nil {
				continue
			}
			parsed = append(parsed, v)
			versionByParsed[v.String()] = raw
		}
		best := semver.Best(sp, parsed)
		if best == nil {
			return Resolved{}, fmt.Errorf("no version matching %q for %s/%s", spec, k, name)
		}
		rawTag := versionByParsed[best.String()]
		return Resolved{Commit: ra.CommitForVersion(rawTag), Version: rawTag}, nil
	}
	return Resolved{}, fmt.Errorf("%s/%s not found on remote", k, name)
}
