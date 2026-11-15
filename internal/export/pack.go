package export

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	regpkg "github.com/satyendrakumarsingh/repocortex/internal/registry"
	"github.com/satyendrakumarsingh/repocortex/internal/security"
	"github.com/satyendrakumarsingh/repocortex/pkg/model"
)

type PackFormat string

const (
	PackFormatXML      PackFormat = "xml"
	PackFormatMarkdown PackFormat = "markdown"
	PackFormatJSON     PackFormat = "json"
)

type PackOptions struct {
	Format       PackFormat
	IncludeTree  bool
	MaxFileBytes int64
	PathPrefix   string
	Redact       bool
}

type PackResult struct {
	Content         string `json:"content,omitempty"`
	FileCount       int    `json:"file_count"`
	TotalBytes      int64  `json:"total_bytes"`
	EstimatedTokens int    `json:"estimated_tokens"`
	Tree            string `json:"tree,omitempty"`
	RedactedCount   int    `json:"redacted_count"`
}

// GeneratePack bundles codebase files into an AI-ready XML, Markdown, or JSON prompt pack.
func GeneratePack(ctx context.Context, reg *regpkg.Registry, runID int64, opts PackOptions) (*PackResult, error) {
	files, err := reg.Files(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("read files from run %d: %w", runID, err)
	}

	maxBytes := opts.MaxFileBytes
	if maxBytes <= 0 {
		maxBytes = 500 * 1024 // 500 KB
	}

	var eligible []model.File
	for _, f := range files {
		if opts.PathPrefix != "" && !strings.HasPrefix(f.Path, opts.PathPrefix) {
			continue
		}
		if f.ParseStatus == model.ParseStatusSkipped && strings.Contains(f.ParseError, "binary") {
			continue
		}
		if f.SizeBytes > maxBytes {
			continue
		}
		eligible = append(eligible, f)
	}

	sort.Slice(eligible, func(i, j int) bool {
		return eligible[i].Path < eligible[j].Path
	})

	var paths []string
	for _, f := range eligible {
		paths = append(paths, f.Path)
	}
	tree := BuildASCIITree(paths)

	type fileItem struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	var packedFiles []fileItem
	var totalBytes int64
	totalRedacted := 0

	for _, f := range eligible {
		// Read full file content through registry excerpt
		content, err := reg.FileExcerpt(ctx, runID, f.Path, 1, 20000)
		if err != nil {
			continue
		}
		if opts.Redact {
			clean, findings := security.Redact(content)
			content = clean
			totalRedacted += len(findings)
		}
		packedFiles = append(packedFiles, fileItem{
			Path:    f.Path,
			Content: content,
		})
		totalBytes += int64(len(content))
	}

	var outContent string
	switch opts.Format {
	case PackFormatJSON:
		payload := map[string]any{
			"tree":             tree,
			"files":            packedFiles,
			"file_count":       len(packedFiles),
			"total_bytes":      totalBytes,
			"estimated_tokens": int(totalBytes / 4),
			"redacted_count":   totalRedacted,
		}
		bytes, _ := json.MarshalIndent(payload, "", "  ")
		outContent = string(bytes)

	case PackFormatMarkdown:
		var sb strings.Builder
		sb.WriteString("# Codebase Context Pack\n\n")
		if opts.IncludeTree {
			sb.WriteString("## Directory Structure\n\n```text\n")
			sb.WriteString(tree)
			sb.WriteString("\n```\n\n")
		}
		sb.WriteString("## Files\n\n")
		for _, pf := range packedFiles {
			sb.WriteString(fmt.Sprintf("### %s\n\n", pf.Path))
			ext := strings.TrimPrefix(filepath.Ext(pf.Path), ".")
			sb.WriteString(fmt.Sprintf("```%s\n%s\n```\n\n", ext, pf.Content))
		}
		outContent = sb.String()

	default: // XML
		var sb strings.Builder
		sb.WriteString("<repocortex_pack>\n")
		if opts.IncludeTree {
			sb.WriteString("  <directory_structure>\n")
			for _, line := range strings.Split(tree, "\n") {
				sb.WriteString("    " + line + "\n")
			}
			sb.WriteString("  </directory_structure>\n")
		}
		sb.WriteString("  <files>\n")
		for _, pf := range packedFiles {
			sb.WriteString(fmt.Sprintf("    <file path=%q>\n", pf.Path))
			sb.WriteString(pf.Content)
			if !strings.HasSuffix(pf.Content, "\n") {
				sb.WriteString("\n")
			}
			sb.WriteString("    </file>\n")
		}
		sb.WriteString("  </files>\n")
		sb.WriteString("</repocortex_pack>\n")
		outContent = sb.String()
	}

	return &PackResult{
		Content:         outContent,
		FileCount:       len(packedFiles),
		TotalBytes:      totalBytes,
		EstimatedTokens: int(totalBytes / 4),
		Tree:            tree,
		RedactedCount:   totalRedacted,
	}, nil
}

// BuildASCIITree constructs an ASCII file hierarchy from a sorted list of relative paths.
func BuildASCIITree(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	type node struct {
		name     string
		children map[string]*node
		isDir    bool
	}

	newNode := func(name string, isDir bool) *node {
		return &node{
			name:     name,
			children: make(map[string]*node),
			isDir:    isDir,
		}
	}

	root := newNode("", true)

	for _, p := range paths {
		parts := strings.Split(filepath.ToSlash(p), "/")
		curr := root
		for i, part := range parts {
			isLast := i == len(parts)-1
			if _, exists := curr.children[part]; !exists {
				curr.children[part] = newNode(part, !isLast)
			}
			curr = curr.children[part]
		}
	}

	var lines []string
	var walk func(n *node, prefix string, isTail bool, isRoot bool)
	walk = func(n *node, prefix string, isTail bool, isRoot bool) {
		if !isRoot {
			connector := "├── "
			if isTail {
				connector = "└── "
			}
			name := n.name
			if n.isDir {
				name += "/"
			}
			lines = append(lines, prefix+connector+name)
		}

		childPrefix := prefix
		if !isRoot {
			if isTail {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}
		}

		// Sort child names
		var names []string
		for k := range n.children {
			names = append(names, k)
		}
		sort.Strings(names)

		for i, k := range names {
			walk(n.children[k], childPrefix, i == len(names)-1, false)
		}
	}

	walk(root, "", true, true)
	return strings.Join(lines, "\n")
}
