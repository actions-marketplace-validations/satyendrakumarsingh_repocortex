package coverage

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// RangeCoverage records a covered line range.
type RangeCoverage struct {
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`
	Count     int `json:"count"`
}

// FileCoverage represents coverage data for a single file.
type FileCoverage struct {
	Path         string          `json:"path"`
	TotalLines   int             `json:"total_lines"`
	CoveredLines int             `json:"covered_lines"`
	CoverageRate float64         `json:"coverage_rate"`
	Lines        map[int]int     `json:"-"`
	Ranges       []RangeCoverage `json:"ranges,omitempty"`
}

// Report holds the parsed test coverage report across all files.
type Report struct {
	Format       string                   `json:"format"`
	Files        map[string]*FileCoverage `json:"files"`
	TotalLines   int                      `json:"total_lines"`
	CoveredLines int                      `json:"covered_lines"`
	CoverageRate float64                  `json:"coverage_rate"`
}

// ParseCoverageFile reads and parses a coverage file, auto-detecting the format.
func ParseCoverageFile(filePath string) (*Report, error) {
	//nolint:gosec // filePath is the user-supplied coverage file to import
	f, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return nil, fmt.Errorf("open coverage file: %w", err)
	}
	defer func() { _ = f.Close() }()

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".xml":
		return ParseCobertura(f)
	case ".info", ".lcov":
		return ParseLCOV(f)
	default:
		// Sniff first line
		r := bufio.NewReader(f)
		header, err := r.Peek(64)
		if err == nil && strings.HasPrefix(string(header), "mode:") {
			return ParseGoCover(r)
		}
		if err == nil && (strings.HasPrefix(string(header), "SF:") || strings.HasPrefix(string(header), "TN:")) {
			return ParseLCOV(r)
		}
		if err == nil && (strings.HasPrefix(strings.TrimSpace(string(header)), "<?xml") || strings.HasPrefix(strings.TrimSpace(string(header)), "<coverage")) {
			return ParseCobertura(r)
		}
		// Fallback to Go coverprofile
		return ParseGoCover(r)
	}
}

// ParseGoCover parses Go's standard coverprofile format.
func ParseGoCover(r io.Reader) (*Report, error) {
	scanner := bufio.NewScanner(r)
	files := make(map[string]*FileCoverage)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		// Format: file.go:startLine.startCol,endLine.endCol numStatements count
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		locParts := strings.Split(parts[0], ":")
		if len(locParts) < 2 {
			continue
		}
		filePath := normalizePath(locParts[0])
		spanParts := strings.Split(locParts[1], ",")
		if len(spanParts) < 2 {
			continue
		}

		startLine, err1 := strconv.Atoi(strings.Split(spanParts[0], ".")[0])
		endLine, err2 := strconv.Atoi(strings.Split(spanParts[1], ".")[0])
		count, err3 := strconv.Atoi(parts[2])
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}

		fc, ok := files[filePath]
		if !ok {
			fc = &FileCoverage{
				Path:  filePath,
				Lines: make(map[int]int),
			}
			files[filePath] = fc
		}

		for l := startLine; l <= endLine; l++ {
			fc.Lines[l] += count
		}
		fc.Ranges = append(fc.Ranges, RangeCoverage{
			StartLine: startLine,
			EndLine:   endLine,
			Count:     count,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read go coverage: %w", err)
	}
	return finalizeReport("go-cover", files), nil
}

// ParseLCOV parses LCOV format (.info / .lcov).
func ParseLCOV(r io.Reader) (*Report, error) {
	scanner := bufio.NewScanner(r)
	files := make(map[string]*FileCoverage)
	var currentFile *FileCoverage

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "SF:"):
			p := normalizePath(strings.TrimPrefix(line, "SF:"))
			currentFile = &FileCoverage{
				Path:  p,
				Lines: make(map[int]int),
			}
			files[p] = currentFile
		case strings.HasPrefix(line, "DA:") && currentFile != nil:
			// DA:<line>,<execution_count>
			daParts := strings.Split(strings.TrimPrefix(line, "DA:"), ",")
			if len(daParts) >= 2 {
				lineNum, _ := strconv.Atoi(daParts[0])
				count, _ := strconv.Atoi(daParts[1])
				if lineNum > 0 {
					currentFile.Lines[lineNum] += count
				}
			}
		case line == "end_of_record":
			currentFile = nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read lcov: %w", err)
	}
	return finalizeReport("lcov", files), nil
}

type xmlCoverage struct {
	Packages []struct {
		Classes []struct {
			Filename string `xml:"filename,attr"`
			Lines    []struct {
				Number int `xml:"number,attr"`
				Hits   int `xml:"hits,attr"`
			} `xml:"lines>line"`
		} `xml:"classes>class"`
	} `xml:"packages>package"`
}

// ParseCobertura parses Cobertura XML format.
func ParseCobertura(r io.Reader) (*Report, error) {
	var xc xmlCoverage
	if err := xml.NewDecoder(r).Decode(&xc); err != nil {
		return nil, fmt.Errorf("decode cobertura xml: %w", err)
	}
	files := make(map[string]*FileCoverage)
	for _, pkg := range xc.Packages {
		for _, cls := range pkg.Classes {
			p := normalizePath(cls.Filename)
			fc, ok := files[p]
			if !ok {
				fc = &FileCoverage{
					Path:  p,
					Lines: make(map[int]int),
				}
				files[p] = fc
			}
			for _, l := range cls.Lines {
				if l.Number > 0 {
					fc.Lines[l.Number] += l.Hits
				}
			}
		}
	}
	return finalizeReport("cobertura", files), nil
}

func finalizeReport(format string, files map[string]*FileCoverage) *Report {
	var totalLines, coveredLines int
	for _, fc := range files {
		var fTotal, fCovered int
		for _, count := range fc.Lines {
			fTotal++
			if count > 0 {
				fCovered++
			}
		}
		fc.TotalLines = fTotal
		fc.CoveredLines = fCovered
		if fTotal > 0 {
			fc.CoverageRate = float64(fCovered) / float64(fTotal)
		}
		totalLines += fTotal
		coveredLines += fCovered
	}

	var rate float64
	if totalLines > 0 {
		rate = float64(coveredLines) / float64(totalLines)
	}

	return &Report{
		Format:       format,
		Files:        files,
		TotalLines:   totalLines,
		CoveredLines: coveredLines,
		CoverageRate: rate,
	}
}

// IsSymbolCovered checks if a symbol range has test coverage.
func (r *Report) IsSymbolCovered(filePath string, startLine, endLine int) (covered bool, coveredCount, totalCount int) {
	norm := normalizePath(filePath)
	fc, ok := r.Files[norm]
	if !ok {
		// Try suffix match
		for p, f := range r.Files {
			if strings.HasSuffix(p, norm) || strings.HasSuffix(norm, p) {
				fc = f
				break
			}
		}
	}
	if fc == nil || len(fc.Lines) == 0 {
		return false, 0, 0
	}

	for l := startLine; l <= endLine; l++ {
		if count, exists := fc.Lines[l]; exists {
			totalCount++
			if count > 0 {
				coveredCount++
			}
		}
	}
	return coveredCount > 0, coveredCount, totalCount
}

// Summary returns a human-readable summary.
func (r *Report) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Coverage Report (%s):\n", r.Format)
	fmt.Fprintf(&b, "Total Coverage: %.1f%% (%d / %d lines covered across %d files)\n",
		r.CoverageRate*100, r.CoveredLines, r.TotalLines, len(r.Files))

	paths := make([]string, 0, len(r.Files))
	for p := range r.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	b.WriteString("\nPer-file breakdown:\n")
	for _, p := range paths {
		fc := r.Files[p]
		fmt.Fprintf(&b, "  - %-40s %5.1f%% (%d/%d lines)\n",
			p, fc.CoverageRate*100, fc.CoveredLines, fc.TotalLines)
	}
	return b.String()
}

func normalizePath(p string) string {
	p = filepath.Clean(p)
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	return p
}
