package mcp

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/satyendrakumarsingh/repocortex/internal/capability/knowledge"
	"github.com/satyendrakumarsingh/repocortex/internal/capability/ownership"
	"github.com/satyendrakumarsingh/repocortex/pkg/model"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// list_knowledge_risks answers "where would losing one person lose most of
// what is known": the directories whose recent commits come from the fewest
// people, most concentrated first.
//
// It reads the same history find_owners does, through the same port, so there
// is one reader of git behind the server and one cache of its walk.
//
// NAMES, NEVER ADDRESSES, for the reason owners.go gives. The capability's
// report has no field that can hold an address, and the name of an author who
// set their git name to their address is cut at the @ before it is returned.

type knowledgeArgs struct {
	Path       string `json:"path,omitempty" jsonschema:"Only measure files under this path prefix, e.g. 'internal/billing/'."`
	Commits    int    `json:"commits,omitempty" jsonschema:"How many recent commits to read. Default 500."`
	Limit      int    `json:"limit,omitempty" jsonschema:"Most directories to return. Default 15."`
	MinCommits int    `json:"min_commits,omitempty" jsonschema:"Fewest commits a directory needs before it is measured. Default 5. Directories below it are counted in the answer, not listed."`
	Depth      int    `json:"depth,omitempty" jsonschema:"Group by the first N path segments instead of by each file's own directory, e.g. 1 for top-level directories. Default 0."`
	Files      bool   `json:"files,omitempty" jsonschema:"Measure each file instead of each directory. Default false."`
}

// defaultKnowledgeItems is how many areas are listed unless asked. One row
// each, so the answer stays a few kilobytes with its notes.
const defaultKnowledgeItems = 15

// knowledgeNameRunes bounds one author name inside a row's reason. A name is
// repository content of any length, and three of them share a bounded line.
const knowledgeNameRunes = 40

func (s *Server) registerKnowledge(allowed map[string]bool) {
	addTool(s, allowed, &sdk.Tool{
		Name:        "list_knowledge_risks",
		Description: toolDescriptions()["list_knowledge_risks"],
	}, s.listKnowledgeRisks)
}

// knowledgeHistory adapts the author-carrying history find_owners reads to
// the knowledge port. The paths are already in the index's path space.
type knowledgeHistory struct{ h ownership.History }

func (a knowledgeHistory) Commits(ctx context.Context, limit int) ([]knowledge.Commit, error) {
	cs, err := a.h.Commits(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]knowledge.Commit, 0, len(cs))
	for _, c := range cs {
		out = append(out, knowledge.Commit{
			SHA: c.SHA, Author: c.AuthorName, AuthorEmail: c.AuthorEmail, When: c.When, Paths: c.Paths,
			ShallowBoundary: c.ShallowBoundary,
		})
	}
	return out, nil
}

func (s *Server) listKnowledgeRisks(ctx context.Context, req *sdk.CallToolRequest, args knowledgeArgs) (*sdk.CallToolResult, *Response, error) {
	if args.Commits < 0 || args.Limit < 0 || args.MinCommits < 0 || args.Depth < 0 {
		return failWith("invalid_argument",
			"Pass commits, limit, min_commits and depth as positive numbers, or omit them.",
			"commits, limit, min_commits and depth must not be negative")
	}
	runID, status, err := s.currentRun(ctx)
	if err != nil {
		return s.noRun(ctx, req, err)
	}
	if s.deps.Owners == nil {
		return s.guide(ctx, req, runID, status,
			"No commit history is available to this server, so knowledge distribution cannot be measured.",
			"This reads git history. The project is not a git repository, or its history could not "+
				"be opened. That is not a finding that knowledge is well spread: nothing was measured. "+
				"Everything else this server answers is unaffected.")
	}

	limit := args.Limit
	if limit == 0 {
		limit = defaultKnowledgeItems
	}
	limit = min(limit, s.limits.MaxItems)

	rep, err := knowledge.Analyze(ctx, knowledgeHistory{s.deps.Owners}, s.deps.Registry, runID, knowledge.Options{
		Limit:        limit,
		CommitWindow: args.Commits,
		MinCommits:   args.MinCommits,
		PathPrefix:   strings.TrimSpace(args.Path),
		Depth:        args.Depth,
		PerFile:      args.Files,
		ShallowClone: s.headRepo != nil && s.headRepo.Shallow(),
	})
	if err != nil {
		return failFrom(err, "knowledge distribution: %v", err)
	}

	resp := &Response{
		IndexRevision: runID,
		IndexStatus:   status,
		Summary:       knowledgeSummary(rep),
	}
	for _, a := range rep.Areas {
		name := path.Base(a.Path)
		if a.Path == knowledge.Root {
			name = "(top directory)"
		}
		resp.Items = append(resp.Items, Item{
			Kind: string(rep.Unit), Name: name, Path: a.Path, Source: string(model.SourceGit),
			Why: knowledgeWhy(a),
		})
	}
	if len(rep.Areas) > 0 {
		resp.Notes = append(resp.Notes,
			"Rows are ordered most concentrated first: the lowest bus factor, then the largest share "+
				"held by one name, then the most commits. 'idle' is the days between that person's last "+
				"change there and the newest commit read. Use find_owners for who has context on one symbol.")
	}
	resp.Notes = append(resp.Notes, rep.Notes...)
	resp.Notes = append(resp.Notes, rep.Caveats...)
	return s.reply(ctx, req, resp)
}

// knowledgeSummary is the one-line answer.
func knowledgeSummary(rep *knowledge.Report) string {
	unit := "directory(ies)"
	if rep.Unit == knowledge.UnitFile {
		unit = "file(s)"
	}
	if rep.CommitsRead == 0 {
		return "No commit history could be read, so knowledge distribution was not measured."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d of %d measured %s listed, most concentrated first, from the last %d commit(s)",
		len(rep.Areas), rep.Measured, unit, rep.CommitsRead)
	if o := rep.Overall; o != nil {
		fmt.Fprintf(&b, " by %d author name(s)", o.Contributors)
	}
	fmt.Fprintf(&b, ". Bus factor of 1: %d of the %d measured.", rep.SolePerson, rep.Measured)
	if rep.Excluded > 0 {
		fmt.Fprintf(&b, " %d more had fewer than %d commits and were not measured.", rep.Excluded, rep.MinCommits)
	}
	if o := rep.Overall; o != nil {
		fmt.Fprintf(&b, " All %d commit(s) in scope taken together: bus factor %d.", o.Commits, o.BusFactor)
	}
	return b.String()
}

// knowledgeWhy is the one-line account of an area's place in the list.
func knowledgeWhy(a knowledge.Area) string {
	var b strings.Builder
	fmt.Fprintf(&b, "bus factor %d: %d commit(s) by %d name(s)", a.BusFactor, a.Commits, a.Contributors)
	for i, c := range a.Top {
		if i == 0 {
			b.WriteString("; ")
		} else {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s %.0f%%", clipRunes(c.Name, knowledgeNameRunes), 100*c.Share)
		if i == 0 && c.LastChanged > 0 {
			fmt.Fprintf(&b, " (last change here %s, idle %dd)",
				time.Unix(c.LastChanged, 0).UTC().Format("2006-01-02"), a.TopIdleDays)
		}
	}
	return b.String()
}

// clipRunes cuts s to at most n runes, marking the cut.
func clipRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
