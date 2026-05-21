package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/niradler/git-skill/internal/git"
	"github.com/niradler/git-skill/internal/lock"
	"github.com/niradler/git-skill/internal/refs"
	"github.com/niradler/git-skill/internal/skill"
)

// version is overridden at build time via -ldflags "-X main.version=v1.2.3".
var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	// When invoked as "git skill", git strips "skill" and passes subcommand.
	// When invoked directly as "git-skill", os.Args[0] is the binary.
	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "init":
		cmdInit(args)
	case "commit":
		cmdCommit(args)
	case "log":
		cmdLog(args)
	case "diff":
		cmdDiff(args)
	case "tag":
		cmdTag(args)
	case "list", "ls":
		cmdList(args)
	case "show":
		cmdShow(args)
	case "push":
		cmdPush(args)
	case "fetch":
		cmdFetch(args)
	case "install":
		cmdInstall(args)
	case "get":
		cmdGet(args)
	case "sync":
		cmdSync(args)
	case "track":
		cmdTrack(args)
	case "version":
		fmt.Printf("git-skill %s\n", version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`git-skill — git-native skill versioning

Named git-skill so git discovers it as "git skill <cmd>".

Commands:
  init    <name> [desc]      Scaffold a new skill and create first commit
  track   <name> <dir>       Import an existing skill directory
  commit  <name> [-m msg]    Snapshot current skill state as a new commit
  log     <name> [-n N]      Show commit history for a skill
  diff    <name> [v1] [v2]   Diff between skill versions
  tag     <name> <version>   Tag a skill release
  list                       List all tracked skills
  show    <name>             Show metadata and tagged versions
  push    <remote>           Push all skills to a remote
  fetch   <remote>           Fetch all skills from a remote
  install <name[@ver]> <dir>            Extract a locally-fetched skill to a dir
  get     <remote> <skill[@ver]> <dir>  Fetch from remote, install, and pin to skill.lock
  sync                                  Reinstall all skills pinned in skill.lock
  version                               Print version
`)
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}

func mustBeRepo() {
	if !git.IsRepo() {
		fatal("not inside a git repository")
	}
}

// skillDir returns the working directory for a skill.
// Convention: .skills/<name> relative to repo root.
func skillDir(name string) string {
	root, err := git.TopLevel()
	if err != nil {
		fatal("cannot find repo root: %v", err)
	}
	return filepath.Join(root, ".skills", name)
}

// installToDir extracts treeHash into target atomically: it writes to
// target+".new", removes any existing target, then renames into place.
// This guarantees old files deleted upstream don't linger after a sync —
// the lockfile's "same bytes on every machine" promise depends on it.
func installToDir(treeHash, target string) error {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}
	tmp := target + ".new"
	// Clean any leftover tmp from a prior aborted run.
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0755); err != nil {
		return fmt.Errorf("mkdir tmp: %w", err)
	}
	if err := git.ReadTreeToDir(treeHash, tmp); err != nil {
		os.RemoveAll(tmp)
		return fmt.Errorf("read-tree: %w", err)
	}
	// Move existing target aside, swap in new, then drop the old.
	// On failure between Rename and RemoveAll, the user can recover from ".old".
	old := target + ".old"
	_ = os.RemoveAll(old)
	if _, err := os.Lstat(target); err == nil {
		if err := os.Rename(target, old); err != nil {
			os.RemoveAll(tmp)
			return fmt.Errorf("stash old: %w", err)
		}
	}
	if err := os.Rename(tmp, target); err != nil {
		// Try to put the old one back.
		if _, statErr := os.Lstat(old); statErr == nil {
			_ = os.Rename(old, target)
		}
		os.RemoveAll(tmp)
		return fmt.Errorf("install: %w", err)
	}
	_ = os.RemoveAll(old)
	return nil
}

// toLockPath returns the canonical lockfile representation of p: relative to
// repo root, using forward slashes. Cross-platform repos depend on this
// (a Windows-authored lockfile must be readable on Linux and vice versa).
// If p escapes root, returns a forward-slashed absolute path as a fallback —
// validateLockfilePath will reject it on use.
func toLockPath(root, p string) string {
	abs := p
	if !filepath.IsAbs(abs) {
		abs, _ = filepath.Abs(p)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return filepath.ToSlash(p)
	}
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

// validateLockfilePath rejects symlink paths from skill.lock that would escape
// the repo root or contain traversal segments. Paths in the lockfile come from
// other people's machines and must not be trusted blindly.
func validateLockfilePath(root, p string) error {
	if p == "" {
		return fmt.Errorf("empty path")
	}
	abs := p
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(root, abs)
	}
	abs = filepath.Clean(abs)
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	rootAbs = filepath.Clean(rootAbs)
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes repo root: %s", p)
	}
	return nil
}

// --- init ---

func cmdInit(args []string) {
	mustBeRepo()
	if len(args) < 1 {
		fatal("usage: git skill init <name> [description]")
	}
	name := args[0]
	if err := refs.ValidateName(name); err != nil {
		fatal("invalid skill name: %v", err)
	}
	desc := "TODO"
	if len(args) > 1 {
		desc = strings.Join(args[1:], " ")
	}

	ref := refs.Ref(name)
	if git.RefExists(ref) {
		fatal("skill %q already exists", name)
	}

	dir := skillDir(name)
	if err := skill.Scaffold(dir, name, desc); err != nil {
		fatal("scaffold: %v", err)
	}

	tree, err := git.WriteTreeFromDir(dir)
	if err != nil {
		fatal("write-tree: %v", err)
	}

	meta, _ := skill.ParseMeta(dir)
	msg := skill.FormatCommitMessage(
		fmt.Sprintf("Initialize skill: %s", name),
		desc,
		meta,
	)

	commit, err := git.CommitTree(tree, msg)
	if err != nil {
		fatal("commit-tree: %v", err)
	}

	if err := git.UpdateRef(ref, commit); err != nil {
		fatal("update-ref: %v", err)
	}

	fmt.Printf("created skill %s\n", name)
	fmt.Printf("  ref:    %s\n", ref)
	fmt.Printf("  commit: %s\n", commit[:12])
	fmt.Printf("  dir:    %s\n", dir)
}

// --- track ---

func cmdTrack(args []string) {
	mustBeRepo()
	if len(args) < 2 {
		fatal("usage: git skill track <name> <dir>")
	}
	name := args[0]
	dir := args[1]

	if err := refs.ValidateName(name); err != nil {
		fatal("invalid skill name: %v", err)
	}

	ref := refs.Ref(name)
	if git.RefExists(ref) {
		fatal("skill %q already tracked", name)
	}

	meta, err := skill.ParseMeta(dir)
	if err != nil {
		fatal("invalid skill directory: %v", err)
	}
	// Spec (SKILL-FORMAT.md) requires name and description in frontmatter.
	// Be strict on track so we never publish a malformed skill.
	if meta == nil || meta.Name == "" || meta.Description == "" {
		fatal("skill at %s is missing required SKILL.md frontmatter (name, description)", dir)
	}

	tree, err := git.WriteTreeFromDir(dir)
	if err != nil {
		fatal("write-tree: %v", err)
	}

	msg := skill.FormatCommitMessage(
		fmt.Sprintf("Track existing skill: %s", name),
		"",
		meta,
	)

	commit, err := git.CommitTree(tree, msg)
	if err != nil {
		fatal("commit-tree: %v", err)
	}

	if err := git.UpdateRef(ref, commit); err != nil {
		fatal("update-ref: %v", err)
	}

	fmt.Printf("tracking %s from %s\n", name, dir)
	fmt.Printf("  ref:    %s\n", ref)
	fmt.Printf("  commit: %s\n", commit[:12])
}

// --- commit ---

func cmdCommit(args []string) {
	mustBeRepo()

	msg := ""
	name := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-m":
			if i+1 < len(args) {
				msg = args[i+1]
				i++
			}
		default:
			if name == "" {
				name = args[i]
			}
		}
	}
	if name == "" {
		fatal("usage: git skill commit <name> [-m message]")
	}
	if err := refs.ValidateName(name); err != nil {
		fatal("invalid skill name: %v", err)
	}
	if msg == "" {
		msg = "Update skill: " + name
	}

	ref := refs.Ref(name)
	if !git.RefExists(ref) {
		fatal("skill %q not tracked — run 'git skill init' or 'git skill track' first", name)
	}

	dir := skillDir(name)
	meta, _ := skill.ParseMeta(dir)

	tree, err := git.WriteTreeFromDir(dir)
	if err != nil {
		fatal("write-tree: %v", err)
	}

	// Read parent state. Used both to short-circuit no-op commits and to
	// CAS the ref update below.
	parentHash, err := git.ResolveRef(ref)
	if err != nil {
		fatal("resolve parent: %v", err)
	}
	parentTree, err := git.Run("rev-parse", parentHash+"^{tree}")
	if err != nil {
		fatal("resolve parent tree: %v", err)
	}
	if tree == parentTree {
		fmt.Println("nothing changed")
		return
	}

	fullMsg := skill.FormatCommitMessage(msg, "", meta)
	commit, err := git.CommitTree(tree, fullMsg, parentHash)
	if err != nil {
		fatal("commit-tree: %v", err)
	}

	// CAS: refuse to update if the ref moved under us. Without the expected
	// old value a racing commit can silently overwrite ours.
	if err := git.UpdateRefCAS(ref, commit, parentHash); err != nil {
		fatal("update-ref: %v (did another commit happen concurrently?)", err)
	}

	fmt.Printf("[%s %s] %s\n", name, commit[:12], msg)
}

// --- log ---

func cmdLog(args []string) {
	mustBeRepo()
	name := ""
	count := 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &count)
				i++
			}
		default:
			if name == "" {
				name = args[i]
			}
		}
	}
	if name == "" {
		fatal("usage: git skill log <name> [-n count]")
	}

	ref := refs.Ref(name)
	if !git.RefExists(ref) {
		fatal("skill %q not found", name)
	}

	format := "%C(yellow)%h%Creset %s %C(dim)(%ar)%Creset %C(blue)<%an>%Creset%n%(trailers:key=Skill-Version,valueonly,separator=%x2c )"
	out, err := git.Log(ref, format, count)
	if err != nil {
		fatal("log: %v", err)
	}
	fmt.Println(out)
}

// --- diff ---

func cmdDiff(args []string) {
	mustBeRepo()
	if len(args) < 1 {
		fatal("usage: git skill diff <name> [v1 v2]")
	}
	name := args[0]
	ref := refs.Ref(name)
	if !git.RefExists(ref) {
		fatal("skill %q not found", name)
	}

	var a, b string
	switch len(args) {
	case 1:
		// Diff HEAD~1 vs HEAD
		a = ref + "~1"
		b = ref
	case 2:
		a = resolveVersion(name, args[1])
		b = ref
	case 3:
		a = resolveVersion(name, args[1])
		b = resolveVersion(name, args[2])
	}

	out, err := git.DiffTree(a, b)
	if err != nil {
		fatal("diff: %v", err)
	}
	if out == "" {
		fmt.Println("no changes")
	} else {
		fmt.Println(out)
	}
}

// resolveVersion turns "v1.0" into the tag ref, or passes through raw refs.
func resolveVersion(name, ver string) string {
	tagRef := refs.TagRef(name, ver)
	if git.RefExists(tagRef) {
		return tagRef
	}
	// Maybe it's a raw commit or relative ref
	return ver
}

// --- tag ---

func cmdTag(args []string) {
	mustBeRepo()
	if len(args) < 2 {
		fatal("usage: git skill tag <name> <version>")
	}
	name := args[0]
	ver := args[1]

	if err := refs.ValidateName(name); err != nil {
		fatal("invalid skill name: %v", err)
	}
	if err := refs.ValidateVersion(ver); err != nil {
		fatal("invalid version: %v", err)
	}

	ref := refs.Ref(name)
	if !git.RefExists(ref) {
		fatal("skill %q not found", name)
	}

	tagRef := refs.TagRef(name, ver)
	if git.RefExists(tagRef) {
		fatal("version %s already exists for %s", ver, name)
	}

	commit, err := git.ResolveRef(ref)
	if err != nil {
		fatal("resolve: %v", err)
	}

	if err := git.UpdateRef(tagRef, commit); err != nil {
		fatal("tag: %v", err)
	}

	v := ver
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	fmt.Printf("tagged %s %s → %s\n", name, v, commit[:12])
}

// --- list ---

func cmdList(_ []string) {
	mustBeRepo()
	format := "%(refname:lstrip=2)\t%(contents:subject)\t%(authordate:relative)"
	lines, err := git.ForEachRef(refs.Prefix, format)
	if err != nil {
		fatal("list: %v", err)
	}
	if len(lines) == 0 {
		fmt.Println("no skills tracked")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "SKILL\tLAST CHANGE\tWHEN\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) == 3 {
			fmt.Fprintf(w, "%s\t%s\t%s\n", parts[0], parts[1], parts[2])
		}
	}
	w.Flush()
}

// --- show ---

func cmdShow(args []string) {
	mustBeRepo()
	if len(args) < 1 {
		fatal("usage: git skill show <name>")
	}
	name := args[0]
	ref := refs.Ref(name)
	if !git.RefExists(ref) {
		fatal("skill %q not found", name)
	}

	// Show commit info
	out, err := git.Log(ref, "commit:  %H%nauthor:  %an <%ae>%ndate:    %ai%nsubject: %s%n%n%b", 1)
	if err != nil {
		fatal("show: %v", err)
	}
	fmt.Println(out)

	tagPattern := refs.TagPrefix + name + "/"
	tags, _ := git.ForEachRef(tagPattern, "%(refname:lstrip=3)")
	if len(tags) > 0 {
		fmt.Println("\nversions:")
		for _, t := range tags {
			fmt.Printf("  %s\n", t)
		}
	}
}

// --- push ---

func cmdPush(args []string) {
	mustBeRepo()
	remote := "origin"
	if len(args) > 0 {
		remote = args[0]
	}
	out, err := git.Run("push", remote, refs.PushRefspec())
	if err != nil {
		fatal("push skills: %v", err)
	}
	if out != "" {
		fmt.Println(out)
	}

	// Also push tags
	out, err = git.Run("push", remote, refs.TagPrefix+"*:"+refs.TagPrefix+"*")
	if err != nil {
		// Not fatal, tags might not exist yet
		fmt.Fprintf(os.Stderr, "warning: skill tags push: %v\n", err)
	}

	fmt.Printf("pushed skills to %s\n", remote)
}

// --- fetch ---

func cmdFetch(args []string) {
	mustBeRepo()
	remote := "origin"
	if len(args) > 0 {
		remote = args[0]
	}
	out, err := git.Run("fetch", remote, refs.FetchRefspec())
	if err != nil {
		fatal("fetch skills: %v", err)
	}
	if out != "" {
		fmt.Println(out)
	}

	// Also fetch tags
	git.Run("fetch", remote, refs.FetchTagRefspec())

	fmt.Printf("fetched skills from %s\n", remote)
}

// --- install ---

func cmdInstall(args []string) {
	mustBeRepo()

	var agentFilter string
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--agent":
			if i+1 < len(args) {
				agentFilter = args[i+1]
				i++
			}
		default:
			positional = append(positional, args[i])
		}
	}
	if len(positional) < 2 {
		fatal("usage: git skill install [--agent <name>] <name[@version]> <dest>")
	}
	spec := positional[0]
	dest := positional[1]

	name, ver, _ := strings.Cut(spec, "@")
	if err := refs.ValidateName(name); err != nil {
		fatal("invalid skill name: %v", err)
	}
	if ver != "" {
		if err := refs.ValidateVersion(ver); err != nil {
			fatal("invalid version: %v", err)
		}
	}

	var ref string
	if ver != "" {
		ref = refs.TagRef(name, ver)
		if !git.RefExists(ref) {
			fatal("version %s not found for %s", ver, name)
		}
	} else {
		ref = refs.Ref(name)
		if !git.RefExists(ref) {
			fatal("skill %q not found", name)
		}
	}

	commit, err := git.ResolveRef(ref)
	if err != nil {
		fatal("resolve: %v", err)
	}
	treeHash, err := git.Run("rev-parse", commit+"^{tree}")
	if err != nil {
		fatal("tree: %v", err)
	}

	// dest is treated as a parent directory; the skill is installed into
	// dest/<name>. This matches user expectations ("install into my skills
	// folder") and keeps multiple skills from colliding in the same dir.
	target := filepath.Join(dest, name)
	if err := installToDir(treeHash, target); err != nil {
		fatal("install: %v", err)
	}

	root, _ := git.TopLevel()
	createAgentSymlinks(root, name, target, agentFilter)

	fmt.Printf("installed %s → %s\n", spec, filepath.ToSlash(target))
}

// --- get ---

func cmdGet(args []string) {
	mustBeRepo()

	var devMode bool
	var agentFilter string
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dev":
			devMode = true
		case "--agent":
			if i+1 < len(args) {
				agentFilter = args[i+1]
				i++
			}
		default:
			positional = append(positional, args[i])
		}
	}
	if len(positional) < 3 {
		fatal("usage: git skill get [--dev] [--agent <name>] <remote> <skill[@version]> <dest>")
	}
	remote := positional[0]
	spec := positional[1]
	dest := positional[2]

	name, ver, _ := strings.Cut(spec, "@")
	if err := refs.ValidateName(name); err != nil {
		fatal("invalid skill name: %v", err)
	}
	if ver != "" {
		if err := refs.ValidateVersion(ver); err != nil {
			fatal("invalid version: %v", err)
		}
	}

	// Fetch skill ref and its tags from the remote.
	// git fetch accepts both remote names and bare URLs.
	fmt.Printf("fetching %s from %s...\n", name, remote)
	skillRef := refs.Ref(name)
	if _, err := git.Run("fetch", remote, skillRef+":"+skillRef); err != nil {
		fatal("fetch skill: %v", err)
	}
	// Tags are best-effort — a skill may not have any yet.
	if _, err := git.Run("fetch", remote, refs.TagPrefix+name+"/*:"+refs.TagPrefix+name+"/*"); err != nil {
		fmt.Fprintf(os.Stderr, "note: no tags found for %s on %s\n", name, remote)
	}

	var ref string
	if ver != "" {
		ref = refs.TagRef(name, ver)
		if !git.RefExists(ref) {
			fatal("version %s not found for %s", ver, name)
		}
	} else {
		ref = skillRef
	}

	commit, err := git.ResolveRef(ref)
	if err != nil {
		fatal("resolve: %v", err)
	}
	treeHash, err := git.Run("rev-parse", commit+"^{tree}")
	if err != nil {
		fatal("tree: %v", err)
	}

	// dest is treated as a parent directory; install into dest/<name>.
	target := filepath.Join(dest, name)
	if err := installToDir(treeHash, target); err != nil {
		fatal("install: %v", err)
	}

	root, err := git.TopLevel()
	if err != nil {
		fatal("repo root: %v", err)
	}
	agents := createAgentSymlinks(root, name, target, agentFilter)

	// --dev means the consumer is also the author and wants to edit the
	// installed skill in place. Skip the lockfile pin so future syncs
	// don't clobber their work.
	if devMode {
		fmt.Printf("installed %s → %s (dev mode, no lockfile entry)\n", spec, target)
		return
	}

	lk, err := lock.Read(root)
	if err != nil {
		fatal("read lockfile: %v", err)
	}
	lk.Skills[name] = lock.Entry{
		Remote:    remote,
		Version:   ver,
		Commit:    commit,
		Canonical: toLockPath(root, target),
		Agents:    agents,
		Dev:       false,
	}
	if err := lk.Write(root); err != nil {
		fatal("write lockfile: %v", err)
	}

	fmt.Printf("installed %s → %s\n", spec, filepath.ToSlash(target))
	fmt.Printf("pinned %s in skill.lock\n", commit[:12])
}

// agentPaths returns the symlink destination paths for each supported agent.
// agentFilter restricts to a single agent when non-empty.
//
// The current set ("claude", "cursor") is intentionally minimal and will grow
// as more agent platforms publish their skill-directory conventions. Extending
// this map is a one-line change — see CONTRIBUTING.md.
func agentPaths(root, skillName, agentFilter string) map[string]string {
	all := map[string]string{
		"claude": filepath.Join(root, ".claude", "skills", skillName),
		"cursor": filepath.Join(root, ".agents", "skills", skillName),
	}
	if agentFilter == "" {
		return all
	}
	if p, ok := all[agentFilter]; ok {
		return map[string]string{agentFilter: p}
	}
	return map[string]string{}
}

// createAgentSymlinks links every supported agent's expected skill path to the
// canonical target. Skips entries that resolve to target itself (which happens
// when the user installs directly into an agent's skills dir).
// Returns the map of agent → link path actually created (or attempted).
func createAgentSymlinks(root, skillName, target, agentFilter string) map[string]string {
	agents := agentPaths(root, skillName, agentFilter)
	created := make(map[string]string, len(agents))
	targetAbs, _ := filepath.Abs(target)
	targetAbs = filepath.Clean(targetAbs)
	for agentName, linkPath := range agents {
		linkAbs, _ := filepath.Abs(linkPath)
		if filepath.Clean(linkAbs) == targetAbs {
			// User installed directly into the agent's skill dir; no link needed.
			continue
		}
		if err := makeSymlink(target, linkPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: symlink for %s: %v\n", agentName, err)
			continue
		}
		created[agentName] = toLockPath(root, linkPath)
	}
	return created
}

// makeSymlink creates a symlink at linkPath pointing to target.
// Skips silently if the correct symlink already exists.
//
// The on-disk symlink target is rewritten relative to the symlink's parent
// directory so it resolves correctly regardless of whether the caller passed
// target as relative-to-cwd or absolute. Without this, `git skill get` with a
// relative install path produces a broken symlink, because OS symlink
// resolution is relative to the symlink's own location, not the cwd.
// A relative-to-parent target also keeps the link valid if the whole repo is
// moved to a new path.
func makeSymlink(target, linkPath string) error {
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	linkDirAbs, err := filepath.Abs(filepath.Dir(linkPath))
	if err != nil {
		return err
	}
	relTarget, err := filepath.Rel(linkDirAbs, targetAbs)
	if err != nil {
		return err
	}
	if existing, err := os.Readlink(linkPath); err == nil {
		if existing == relTarget {
			return nil
		}
		os.Remove(linkPath)
	} else if _, err := os.Lstat(linkPath); err == nil {
		return fmt.Errorf("%s exists and is not a symlink", linkPath)
	}
	return os.Symlink(relTarget, linkPath)
}

// --- sync ---

func cmdSync(_ []string) {
	mustBeRepo()

	root, err := git.TopLevel()
	if err != nil {
		fatal("repo root: %v", err)
	}
	lk, err := lock.Read(root)
	if err != nil {
		fatal("read lockfile: %v", err)
	}
	if len(lk.Skills) == 0 {
		fmt.Println("skill.lock is empty — nothing to sync")
		return
	}

	for name, entry := range lk.Skills {
		if err := refs.ValidateName(name); err != nil {
			fatal("invalid skill name %q in skill.lock: %v", name, err)
		}
		if entry.Dev {
			fmt.Printf("skipped %s (dev mode)\n", name)
			continue
		}

		canonical := filepath.FromSlash(entry.Canonical)
		if canonical == "" {
			canonical = filepath.Join(root, ".skills", name)
		}
		if err := validateLockfilePath(root, canonical); err != nil {
			fatal("invalid canonical path for %s: %v", name, err)
		}

		treeHash, treeErr := git.Run("rev-parse", entry.Commit+"^{tree}")
		if treeErr != nil {
			// Object not in local store — fetch the ref first, then resolve.
			// If the upstream ref no longer reaches the pinned commit, the
			// second rev-parse will fail loudly (which is the right outcome:
			// the lock points at a commit the remote can no longer produce).
			fmt.Printf("fetching %s from %s...\n", name, entry.Remote)
			skillRef := refs.Ref(name)
			if _, err := git.Run("fetch", entry.Remote, skillRef+":"+skillRef); err != nil {
				fatal("fetch %s: %v", name, err)
			}
			treeHash, err = git.Run("rev-parse", entry.Commit+"^{tree}")
			if err != nil {
				fatal("pinned commit %s for %s is not reachable from %s — was the tag force-moved upstream?",
					entry.Commit[:12], name, entry.Remote)
			}
		}

		if err := installToDir(treeHash, canonical); err != nil {
			fatal("install %s: %v", name, err)
		}

		// Recreate agent symlinks from the lock, but treat lockfile-provided
		// paths as untrusted: validate each one stays inside the repo, and
		// skip self-symlinks.
		agents := entry.Agents
		if len(agents) == 0 {
			agents = agentPaths(root, name, "")
		}
		canonicalAbs, _ := filepath.Abs(canonical)
		canonicalAbs = filepath.Clean(canonicalAbs)
		for agentName, linkPath := range agents {
			linkPath = filepath.FromSlash(linkPath)
			if err := validateLockfilePath(root, linkPath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: skipping symlink for %s: %v\n", agentName, err)
				continue
			}
			linkAbs, _ := filepath.Abs(linkPath)
			if filepath.Clean(linkAbs) == canonicalAbs {
				continue
			}
			if err := makeSymlink(canonical, linkPath); err != nil {
				fmt.Fprintf(os.Stderr, "warning: symlink for %s: %v\n", agentName, err)
			}
		}

		fmt.Printf("synced %s → %s (%s)\n", name, filepath.ToSlash(canonical), entry.Commit[:12])
	}

	fmt.Printf("\n%d skill(s) processed from skill.lock\n", len(lk.Skills))
}
