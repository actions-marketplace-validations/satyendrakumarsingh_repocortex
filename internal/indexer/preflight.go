package indexer

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// projectBlock returns why a C/C++ or Rust project cannot be indexed however
// the machine is set up, or "" when nothing is known against it.
//
// What is read here can only BLOCK a project. A Cargo.toml is opened for one
// fact, whether it declares a workspace, and the answer never changes an
// argument, a path or an environment variable: it can turn a run into a
// reported refusal and nothing else.
func projectBlock(repoRoot string, p Project) string {
	switch p.Language {
	case LangCPP:
		if p.hasMarker("compile_commands.json") {
			return ""
		}
		return "scip-clang reads a compilation database and this project has no compile_commands.json " +
			"beside its " + strings.Join(p.Markers, ", ") + ", so it cannot run. Generate one and copy it " +
			"into " + dirLabel(p.Dir) + " (a copy, because a symbolic link is not trusted as a marker): " +
			"CMake: cmake -S . -B build -DCMAKE_EXPORT_COMPILE_COMMANDS=ON && cp build/compile_commands.json " +
			"<project dir>; Meson writes it into its build directory; for make, run the build under bear"
	case LangRust:
		if hasCargoWorkspace(filepath.Join(repoRoot, filepath.FromSlash(p.Dir))) {
			return "this is a Cargo workspace, and RepoCortex indexes a single Cargo package only: its crates " +
				"are not mapped to their paths here, so its member crates are not indexed either. " +
				"To index it anyway, run `rust-analyzer scip . --output index.scip` yourself in " +
				dirLabel(p.Dir) + " and import it with `repocortex scip import index.scip`"
		}
	}
	return ""
}

func dirLabel(dir string) string {
	if dir == "." || dir == "" {
		return "the repository root"
	}
	return dir
}

// hasCargoWorkspace reports whether dir/Cargo.toml has a [workspace] table.
// Textual and bounded, like requiredRelease: a table header at the start of a
// line, in the first megabyte.
func hasCargoWorkspace(dir string) bool {
	f, err := os.Open(filepath.Join(dir, "Cargo.toml")) //nolint:gosec // G304: project dir the user named + a fixed filename
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	read := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		read += len(line)
		if read > 1<<20 {
			return false
		}
		if line == "[workspace]" || strings.HasPrefix(line, "[workspace.") {
			return true
		}
	}
	return false
}

// cargoWorkspaceAbove reports whether p lies beneath another detected Rust
// project whose Cargo.toml declares a workspace: p is then one of its members.
func cargoWorkspaceAbove(repoRoot string, projects []Project, p Project) bool {
	for _, o := range projects {
		if o.Language != LangRust || o.Dir == p.Dir {
			continue
		}
		if o.Dir != "." && !strings.HasPrefix(p.Dir, o.Dir+"/") {
			continue
		}
		if hasCargoWorkspace(filepath.Join(repoRoot, filepath.FromSlash(path.Clean(o.Dir)))) {
			return true
		}
	}
	return false
}
