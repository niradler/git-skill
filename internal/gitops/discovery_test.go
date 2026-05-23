package gitops

import (
	"reflect"
	"sort"
	"testing"

	"github.com/niradler/git-skill/internal/kind"
)

func TestParseLsRemote(t *testing.T) {
	in := []string{
		"abc123\trefs/assets/skill/acme/api-conventions",
		"def456\trefs/assets/agent/nir/reviewer",
		"111111\trefs/asset-tags/skill/acme/api-conventions/v1.0.0",
		"222222\trefs/asset-tags/skill/acme/api-conventions/v1.1.0",
		"333333\trefs/heads/main",
	}
	got, err := parseLsRemote(in)
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(got, func(i, j int) bool { return got[i].Name < got[j].Name })
	want := []RemoteAsset{
		{Kind: kind.Skill, Name: "acme/api-conventions", Commit: "abc123",
			Versions: []string{"v1.0.0", "v1.1.0"},
			Commits:  map[string]string{"v1.0.0": "111111", "v1.1.0": "222222"}},
		{Kind: kind.Agent, Name: "nir/reviewer", Commit: "def456"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got = %+v\nwant = %+v", got, want)
	}
}
