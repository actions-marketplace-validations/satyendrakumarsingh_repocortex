package pipeline

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/satyendrakumarsingh/repocortex/internal/indexer"
)

var vueScriptBlock = regexp.MustCompile(`(?is)<script((?:"[^"]*"|'[^']*'|[^>"'])*)>(.*?)</script>`)

// transformVueToTS extracts the script blocks from a Vue Single File Component (.vue),
// replacing template and markup with whitespace while preserving the exact line numbers
// and byte offsets of the script contents so that compiler indexers (scip-typescript)
// generate occurrences and definitions that align 1:1 with the original .vue file coordinates.
func transformVueToTS(src []byte) ([]byte, bool) {
	matches := vueScriptBlock.FindAllSubmatchIndex(src, -1)
	if len(matches) == 0 {
		return nil, false
	}

	out := make([]byte, len(src))
	copy(out, src)

	lastEnd := 0
	for _, m := range matches {
		fullStart, fullEnd := m[0], m[1]
		scriptBodyStart, scriptBodyEnd := m[4], m[5]

		// Mask everything between previous script and this script tag
		maskNonScript(out[lastEnd:fullStart])

		// Mask the <script ...> open tag itself (from fullStart to scriptBodyStart)
		maskNonScript(out[fullStart:scriptBodyStart])

		// Mask </script> close tag (from scriptBodyEnd to fullEnd)
		maskNonScript(out[scriptBodyEnd:fullEnd])

		lastEnd = fullEnd
	}
	// Mask any trailing template or style after the last script
	if lastEnd < len(out) {
		maskNonScript(out[lastEnd:])
	}

	return out, true
}

func maskNonScript(b []byte) {
	for i := range b {
		if b[i] != '\n' && b[i] != '\r' {
			b[i] = ' '
		}
	}
}

// stageVueMirrors writes temporary .vue.ts mirror files alongside .vue files in
// TypeScript projects before running scip-typescript, enabling compiler ground truth
// for Vue Single File Components. Returns the list of created mirror paths for cleanup.
func stageVueMirrors(root string, projects []indexer.Project) []string {
	var out []string
	for _, p := range projects {
		if p.Language != indexer.LangTypeScript {
			continue
		}
		projDir := filepath.Join(root, filepath.FromSlash(p.Dir))
		_ = filepath.WalkDir(projDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil //nolint:nilerr // unreadable file/dir is skipped in staging
			}
			if d.IsDir() {
				name := d.Name()
				if name == "node_modules" || name == ".nuxt" || name == ".output" || name == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".vue") {
				return nil
			}
			src, err := os.ReadFile(path) //nolint:gosec // staging Vue mirrors in project dir
			if err != nil || len(src) == 0 {
				return nil //nolint:nilerr // unreadable Vue file is skipped in staging
			}
			tsSrc, ok := transformVueToTS(src)
			if !ok {
				return nil
			}
			mirrorPath := path + ".ts"
			if err := os.WriteFile(mirrorPath, tsSrc, 0o600); err == nil { //nolint:gosec // staging Vue mirrors in project dir
				out = append(out, mirrorPath)
			}
			return nil
		})
	}
	return out
}
