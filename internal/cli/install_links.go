package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
)

// Deeplink represents a 1-click installation URI for an AI coding agent or editor.
type Deeplink struct {
	Client      string
	Description string
	URI         string
}

// AgentDeeplinks returns 1-click MCP install links for supported editors.
func AgentDeeplinks(root string) []Deeplink {
	argsJSON, _ := json.Marshal([]string{"--project", root, "serve"})
	encodedArgs := url.QueryEscape(string(argsJSON))

	return []Deeplink{
		{
			Client:      "Cursor",
			Description: "One-click add to Cursor MCP settings",
			URI:         fmt.Sprintf("cursor://anysphere.cursor-always-local/mcp/add?name=repocortex&command=repocortex&args=%s", encodedArgs),
		},
		{
			Client:      "VS Code",
			Description: "One-click install into VS Code Copilot/MCP",
			URI:         fmt.Sprintf("vscode:mcp/install?name=repocortex&command=repocortex&args=%s", encodedArgs),
		},
		{
			Client:      "VS Code Insiders",
			Description: "One-click install into VS Code Insiders",
			URI:         fmt.Sprintf("vscode-insiders:mcp/install?name=repocortex&command=repocortex&args=%s", encodedArgs),
		},
		{
			Client:      "Windsurf",
			Description: "One-click install into Windsurf / Codeium",
			URI:         fmt.Sprintf("windsurf://mcp/install?name=repocortex&command=repocortex&args=%s", encodedArgs),
		},
		{
			Client:      "LM Studio",
			Description: "One-click connect in LM Studio MCP catalog",
			URI:         fmt.Sprintf("lmstudio://mcp/install?name=repocortex&command=repocortex&args=%s", encodedArgs),
		},
	}
}

// printInstallLinks prints clickable terminal markdown links and URIs.
func printInstallLinks(w io.Writer, root string) {
	_, _ = fmt.Fprintln(w, "1-Click Installation Links for Coding Agents:")
	for _, dl := range AgentDeeplinks(root) {
		_, _ = fmt.Fprintf(w, "\n  %s (%s):\n", dl.Client, dl.Description)
		_, _ = fmt.Fprintf(w, "    %s\n", dl.URI)
	}
	_, _ = fmt.Fprintln(w, "\nClicking or opening these links in your browser/terminal triggers the respective editor's MCP registration flow.")
}
