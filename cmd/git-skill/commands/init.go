package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/niradler/git-skill/internal/runtimes"
	"github.com/niradler/git-skill/internal/state"
)

const gitignoreBlockMarker = "# >>> git-skill managed (do not edit between markers) >>>"
const gitignoreBlockEnd = "# <<< git-skill managed <<<"

func Init(p Profile, args []string, stdout, stderr io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	st, err := state.Read(cwd)
	if err != nil {
		return fmt.Errorf("read existing %s: %w", state.Filename, err)
	}
	if _, err := os.Stat(filepath.Join(cwd, state.Filename)); os.IsNotExist(err) {
		if err := st.Write(cwd); err != nil {
			return fmt.Errorf("write %s: %w", state.Filename, err)
		}
		fmt.Fprintf(stdout, "Created %s\n", state.Filename)
	}

	if err := ensureGitignoreBlock(cwd, st); err != nil {
		return fmt.Errorf("update .gitignore: %w", err)
	}
	return nil
}

func ensureGitignoreBlock(repoRoot string, st *state.State) error {
	gi := filepath.Join(repoRoot, ".gitignore")
	existing, _ := os.ReadFile(gi)
	if strings.Contains(string(existing), gitignoreBlockMarker) {
		return nil
	}
	block := buildGitignoreBlock(st)
	out := string(existing)
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	out += block
	return os.WriteFile(gi, []byte(out), 0644)
}

func buildGitignoreBlock(st *state.State) string {
	var b strings.Builder
	b.WriteString(gitignoreBlockMarker + "\n")
	b.WriteString("# Canonical asset roots - checked-in metadata, ignored content.\n")
	b.WriteString("/" + st.Config.SkillsRoot + "/\n")
	b.WriteString("/" + st.Config.AgentsRoot + "/\n")
	b.WriteString("# Runtime fan-out directories (derived from runtimes.Registry).\n")
	for _, line := range runtimes.GitignoreLines() {
		b.WriteString(line + "\n")
	}
	b.WriteString(gitignoreBlockEnd + "\n")
	return b.String()
}
