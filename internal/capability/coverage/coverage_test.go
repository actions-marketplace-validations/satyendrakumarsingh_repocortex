package coverage

import (
	"strings"
	"testing"
)

func TestParseGoCover(t *testing.T) {
	data := `mode: set
github.com/example/pkg/service.go:10.2,15.10 2 1
github.com/example/pkg/service.go:16.2,20.10 1 0
github.com/example/pkg/handler.go:5.2,8.15 3 5
`
	rep, err := ParseGoCover(strings.NewReader(data))
	if err != nil {
		t.Fatalf("ParseGoCover: %v", err)
	}

	if rep.Format != "go-cover" {
		t.Errorf("format = %q, want go-cover", rep.Format)
	}
	if len(rep.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(rep.Files))
	}

	cov, covLines, totLines := rep.IsSymbolCovered("github.com/example/pkg/service.go", 10, 15)
	if !cov || covLines == 0 {
		t.Errorf("expected lines 10-15 to be covered, got covered=%v (%d/%d)", cov, covLines, totLines)
	}

	covUncovered, _, _ := rep.IsSymbolCovered("github.com/example/pkg/service.go", 16, 20)
	if covUncovered {
		t.Errorf("expected lines 16-20 to be uncovered")
	}
}

func TestParseLCOV(t *testing.T) {
	data := `TN:
SF:src/index.js
DA:1,1
DA:2,1
DA:3,0
DA:4,2
end_of_record
`
	rep, err := ParseLCOV(strings.NewReader(data))
	if err != nil {
		t.Fatalf("ParseLCOV: %v", err)
	}

	if rep.Format != "lcov" {
		t.Errorf("format = %q, want lcov", rep.Format)
	}
	fc, ok := rep.Files["src/index.js"]
	if !ok {
		t.Fatalf("file src/index.js not found in report")
	}
	if fc.TotalLines != 4 || fc.CoveredLines != 3 {
		t.Errorf("expected 3/4 lines, got %d/%d", fc.CoveredLines, fc.TotalLines)
	}
}

func TestParseCobertura(t *testing.T) {
	data := `<?xml version="1.0" ?>
<coverage version="5.5">
  <packages>
    <package name="pkg">
      <classes>
        <class name="calc.py" filename="pkg/calc.py">
          <lines>
            <line number="1" hits="1"/>
            <line number="2" hits="0"/>
            <line number="3" hits="1"/>
          </lines>
        </class>
      </classes>
    </package>
  </packages>
</coverage>
`
	rep, err := ParseCobertura(strings.NewReader(data))
	if err != nil {
		t.Fatalf("ParseCobertura: %v", err)
	}

	if rep.Format != "cobertura" {
		t.Errorf("format = %q, want cobertura", rep.Format)
	}
	fc, ok := rep.Files["pkg/calc.py"]
	if !ok {
		t.Fatalf("file pkg/calc.py not found in report")
	}
	if fc.TotalLines != 3 || fc.CoveredLines != 2 {
		t.Errorf("expected 2/3 lines, got %d/%d", fc.CoveredLines, fc.TotalLines)
	}
}
