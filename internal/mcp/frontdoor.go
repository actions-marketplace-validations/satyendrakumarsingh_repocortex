package mcp

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type frontdoorArgs struct {
	Action string `json:"action" jsonschema:"The code intelligence action to execute: 'explore' (single-turn deep dive on a symbol with code, callers, callees, blast-radius and routes), 'context' (task-oriented context pack), 'search' (code search), 'symbol' (find/get symbol definitions), 'callers' (find upstream callers), 'callees' (find downstream callees), 'references' (find usage references), 'impact' (blast-radius impact analysis), 'tests' (find tests covering a symbol), 'diff' (compress git diff), 'overview' (project overview), 'query' (read-only declarative graph query)."`

	Query string `json:"query,omitempty" jsonschema:"Search query or natural language task description."`
	Name  string `json:"name,omitempty" jsonschema:"Symbol name to look up or analyze."`
	Path  string `json:"path,omitempty" jsonschema:"File path to disambiguate or scope the query."`
	Line  int    `json:"line,omitempty" jsonschema:"1-based line number for precise symbol selection."`

	Diff   string `json:"diff,omitempty" jsonschema:"Unified git diff text for impact or diff compression."`
	Within string `json:"within,omitempty" jsonschema:"Path filter for call sites in callers or references (e.g. 'src/')."`
	Kind   string `json:"kind,omitempty" jsonschema:"Optional symbol kind filter: class, interface, function, method, field, constant."`

	MinConfidence string `json:"min_confidence,omitempty" jsonschema:"Confidence floor: exact, high, medium, or low."`
	Limit         int    `json:"limit,omitempty" jsonschema:"Maximum items to return."`
	Offset        int    `json:"offset,omitempty" jsonschema:"Pagination offset."`
}

func (s *Server) registerFrontdoor(allowed map[string]bool) {
	if allowed == nil || !allowed["repocortex"] {
		return
	}
	descs := toolDescriptions()
	addTool(s, allowed, &sdk.Tool{
		Name:        "repocortex",
		Description: descs["repocortex"],
	}, s.repocortexFrontdoor)
}

func (s *Server) repocortexFrontdoor(ctx context.Context, req *sdk.CallToolRequest, args frontdoorArgs) (*sdk.CallToolResult, *Response, error) {
	action := strings.ToLower(strings.TrimSpace(args.Action))
	if action == "" {
		return failWith("invalid_argument",
			"Specify an action: 'context', 'search', 'symbol', 'callers', 'callees', 'references', 'impact', 'tests', 'diff', 'overview', or 'query'.",
			"action is required")
	}

	switch action {
	case "explore", "inspect":
		name := args.Name
		if name == "" {
			name = args.Query
		}
		return s.exploreSymbol(ctx, req, exploreArgs{
			Name:          name,
			Path:          args.Path,
			Line:          args.Line,
			MinConfidence: args.MinConfidence,
			Limit:         args.Limit,
		})

	case "context", "task", "task_context", "get_task_context":
		if s.deps.Graph == nil {
			return failWith("unsupported",
				"Resolve the project graph first with 'repocortex resolve' to enable task context.",
				"context requires resolved graph")
		}
		task := args.Query
		if task == "" {
			task = args.Name
		}
		return s.getTaskContext(ctx, req, taskContextArgs{
			Task:             task,
			MaxFiles:         args.Limit,
			minConfidenceArg: minConfidenceArg{MinConfidence: args.MinConfidence},
		})

	case "search", "search_code":
		q := args.Query
		if q == "" {
			q = args.Name
		}
		return s.searchCode(ctx, req, searchArgs{
			Query: q,
			Limit: args.Limit,
		})

	case "symbol", "find_symbol", "get_symbol":
		name := args.Name
		if name == "" {
			name = args.Query
		}
		if args.Path != "" || args.Line > 0 {
			return s.getSymbol(ctx, req, symbolRefArgs{
				Name:             name,
				Path:             args.Path,
				lineArg:          lineArg{Line: args.Line},
				minConfidenceArg: minConfidenceArg{MinConfidence: args.MinConfidence},
			})
		}
		return s.findSymbol(ctx, req, symbolArgs{
			Name:             name,
			Kind:             args.Kind,
			minConfidenceArg: minConfidenceArg{MinConfidence: args.MinConfidence},
		})

	case "callers", "find_callers":
		name := args.Name
		if name == "" {
			name = args.Query
		}
		return s.findCallers(ctx, req, siteArgs{
			relationArgs: relationArgs{
				Name:             name,
				Path:             args.Path,
				lineArg:          lineArg{Line: args.Line},
				minConfidenceArg: minConfidenceArg{MinConfidence: args.MinConfidence},
				pageArg:          pageArg{Limit: args.Limit, Offset: args.Offset},
			},
			Within: args.Within,
		})

	case "callees", "find_callees":
		name := args.Name
		if name == "" {
			name = args.Query
		}
		return s.findCallees(ctx, req, relationArgs{
			Name:             name,
			Path:             args.Path,
			lineArg:          lineArg{Line: args.Line},
			minConfidenceArg: minConfidenceArg{MinConfidence: args.MinConfidence},
			pageArg:          pageArg{Limit: args.Limit, Offset: args.Offset},
		})

	case "references", "refs", "find_references":
		name := args.Name
		if name == "" {
			name = args.Query
		}
		return s.findReferences(ctx, req, siteArgs{
			relationArgs: relationArgs{
				Name:             name,
				Path:             args.Path,
				lineArg:          lineArg{Line: args.Line},
				minConfidenceArg: minConfidenceArg{MinConfidence: args.MinConfidence},
				pageArg:          pageArg{Limit: args.Limit, Offset: args.Offset},
			},
			Within: args.Within,
		})

	case "impact", "analyze_impact":
		if s.deps.Graph == nil {
			return failWith("unsupported",
				"Resolve the project graph first with 'repocortex resolve' to enable impact analysis.",
				"impact requires resolved graph")
		}
		name := args.Name
		if name == "" {
			name = args.Query
		}
		return s.impactAnalysis(ctx, req, impactArgs{
			Symbol:        name,
			Path:          args.Path,
			lineArg:       lineArg{Line: args.Line},
			MinConfidence: args.MinConfidence,
			pageArg:       pageArg{Limit: args.Limit, Offset: args.Offset},
		})

	case "tests", "find_tests":
		if s.deps.Graph == nil {
			return failWith("unsupported",
				"Resolve the project graph first with 'repocortex resolve' to enable test discovery.",
				"tests requires resolved graph")
		}
		name := args.Name
		if name == "" {
			name = args.Query
		}
		return s.findTests(ctx, req, testsArgs{
			targetArgs: targetArgs{
				Symbol:           name,
				Path:             args.Path,
				lineArg:          lineArg{Line: args.Line},
				minConfidenceArg: minConfidenceArg{MinConfidence: args.MinConfidence},
				pageArg:          pageArg{Limit: args.Limit, Offset: args.Offset},
			},
		})

	case "diff", "minimize_diff":
		return s.minimizeDiff(ctx, req, minimizeDiffArgs{
			Diff: args.Diff,
		})

	case "overview", "project_overview", "get_project_overview":
		return s.getProjectOverview(ctx, req, emptyArgs{})

	case "query", "sql", "graph_query", "query_graph", "search_graph":
		q := args.Query
		if q == "" {
			q = args.Name
		}
		return s.searchGraph(ctx, req, searchGraphArgs{
			Query: q,
			Limit: args.Limit,
		})

	default:
		return failWith("invalid_argument",
			fmt.Sprintf("Unknown action %q. Supported actions: 'context', 'search', 'symbol', 'callers', 'callees', 'references', 'impact', 'tests', 'diff', 'overview', 'query'.", action),
			"unknown action: %s", action)
	}
}
