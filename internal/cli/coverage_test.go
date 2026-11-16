package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestCoverageImportCmd(t *testing.T) {
	tmpDir := t.TempDir()
	covFile := filepath.Join(tmpDir, "coverage.out")
	covContent := `mode: set
github.com/example/pkg/service.go:10.2,15.10 2 1
github.com/example/pkg/service.go:16.2,20.10 1 0
`
	if err := os.WriteFile(covFile, []byte(covContent), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	g := &Global{}
	cmd := newCoverageCmd(g)
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"import", covFile})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute coverage import: %v", err)
	}
}
