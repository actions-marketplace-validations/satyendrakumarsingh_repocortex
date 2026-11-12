package mcp

import (
	"encoding/json"
	"io"
	"sync"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---------------------------------------------------------------- audit log
//
// WHO ASKED THE INDEX WHAT, AND WHEN.
//
// An agent session reads the structure of a private codebase. A team that has
// to answer "what did the agent look at before it made this change?" needs a
// record that does not depend on the agent's own transcript, and a security
// review needs one before it will approve the tool at all.
//
// The log is opt-in and local: one JSON object per line, appended to a file
// the user names, never sent anywhere. It records the call, not the answer —
// the answer is repository content, and copying it into a second file would
// double the places private source can leak from.

// AuditEvent is one tool call.
type AuditEvent struct {
	Time string `json:"time"` // RFC 3339, UTC
	// Session distinguishes concurrent agents on one server. It is the
	// transport's session id when there is one (HTTP), otherwise empty.
	Session string `json:"session,omitempty"`
	// Transport is how the call arrived: "mcp" for a protocol client, or the
	// name of an in-process front end ("rest", "cli", "portal").
	Transport string `json:"transport"`
	Tool      string `json:"tool"`
	// Arguments are the call's arguments as sent, cut at auditArgBytes.
	Arguments  json.RawMessage `json:"arguments,omitempty"`
	ArgsCut    bool            `json:"arguments_truncated,omitempty"`
	DurationMS int64           `json:"duration_ms"`
	// Outcome is "ok", "tool_error" (the tool answered with isError) or
	// "protocol_error" (the call never reached a handler).
	Outcome string `json:"outcome"`
	// ResultBytes is the size of the answer's text. StructuredBytes is the
	// size of the structured copy of the same answer. Both are sizes only:
	// the answer itself is never recorded.
	ResultBytes     int64 `json:"result_bytes"`
	StructuredBytes int64 `json:"structured_bytes,omitempty"`
	// CitedFiles and CitedFileBytes are how many indexed files this answer
	// named that no earlier answer in the same session had named, and their
	// total size when indexed. Numbers only; the paths are not recorded.
	// Summed over a session they are the size of the distinct files its
	// answers pointed at, which is an upper bound on reading those files whole
	// and not a saving. See usage.go.
	CitedFiles     int64 `json:"cited_files,omitempty"`
	CitedFileBytes int64 `json:"cited_file_bytes,omitempty"`
}

// auditArgBytes bounds the arguments kept per event. A task description or a
// pasted diff can be large, and an audit log that grows by a megabyte per call
// is one that gets turned off.
const auditArgBytes = 2048

// auditToolBytes bounds the tool name kept per event. A registered name is a
// few words; a longer one is something a client invented, and the name is the
// one field of an event the client writes in full.
const auditToolBytes = 128

// AuditLog appends events to a writer, one JSON line each.
//
// Safe for concurrent use: an HTTP server answers several sessions at once and
// interleaved partial lines would make the file unparseable.
type AuditLog struct {
	mu sync.Mutex
	w  io.Writer
}

// NewAuditLog returns a log writing to w. The caller owns w and closes it.
func NewAuditLog(w io.Writer) *AuditLog {
	return &AuditLog{w: w}
}

// Record writes one event. A write failure is dropped: refusing to answer an
// agent because a log disk is full would turn an observability feature into an
// outage, and the caller has no better place to report it.
func (a *AuditLog) Record(ev AuditEvent) {
	if a == nil || a.w == nil {
		return
	}
	line, err := json.Marshal(ev)
	if err != nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	_, _ = a.w.Write(append(line, '\n'))
}

// auditEvent builds the record of one measured call. The measuring is done
// once, in usageMiddleware, and shared with the session counters.
func (s *Server) auditEvent(call *sdk.CallToolRequest, m callMeasure, scratch *callScratch) AuditEvent {
	ev := AuditEvent{
		Time:            m.started.UTC().Format(time.RFC3339),
		Transport:       s.transportOf(call),
		Tool:            clip(m.tool, auditToolBytes),
		DurationMS:      m.elapsed.Milliseconds(),
		Outcome:         m.outcome,
		ResultBytes:     m.resultBytes,
		StructuredBytes: m.structuredBytes,
	}
	if scratch != nil {
		ev.CitedFiles, ev.CitedFileBytes = scratch.citedFiles, scratch.citedFileBytes
	}
	if call.Session != nil {
		ev.Session = call.Session.ID()
	}
	if args := call.Params.Arguments; len(args) > 0 {
		if len(args) > auditArgBytes {
			// A cut JSON document is not JSON, so it is stored as a string.
			cut, _ := json.Marshal(string(args[:auditArgBytes]))
			ev.Arguments, ev.ArgsCut = cut, true
		} else if json.Valid(args) {
			ev.Arguments = args
		}
	}
	return ev
}

// transportOf names the front end a call arrived through.
func (s *Server) transportOf(call *sdk.CallToolRequest) string {
	if call != nil && call.Session != nil {
		if name, ok := s.frontEnds.Load(call.Session); ok {
			return name.(string)
		}
	}
	return "mcp"
}
