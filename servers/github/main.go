// mcpfs-github: GitHub MCP resource server for mcpfs.
// Uses mcpserve framework. Speaks MCP JSON-RPC over stdio.
//
// Resources:
//   github://repos                              - user's repos (slim)
//   github://repos/{owner}/{repo}               - repo details
//   github://repos/{owner}/{repo}/issues        - open issues (slim)
//   github://repos/{owner}/{repo}/pulls         - open PRs (slim)
//   github://repos/{owner}/{repo}/readme        - README content
//   github://repos/{owner}/{repo}/actions       - recent workflow runs
//   github://repos/{owner}/{repo}/releases      - releases
//   github://notifications                      - unread notifications
//   github://gists                              - user's gists
//
// Auth: GITHUB_TOKEN env var, or `gh auth token` fallback.
package main

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/airshelf/mcpfs/pkg/mcpserve"
	"github.com/airshelf/mcpfs/pkg/mcptool"
)

//go:embed tools.json
var toolSchemas []byte

var token string

func init() {
	token = os.Getenv("GITHUB_TOKEN")
	if token == "" {
		out, err := exec.Command("gh", "auth", "token").Output()
		if err == nil {
			token = strings.TrimSpace(string(out))
		}
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "mcpfs-github: set GITHUB_TOKEN or install gh CLI")
		os.Exit(1)
	}
}

func mcpURL() string {
	if u := os.Getenv("GITHUB_MCP_URL"); u != "" {
		return u
	}
	return "https://api.githubcopilot.com/mcp/"
}

func ghAPI(path string) (json.RawMessage, error) {
	url := "https://api.github.com" + path
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GitHub API %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}
	return json.RawMessage(body), nil
}

// slimObjects extracts only the named fields from an array of JSON objects.
func slimObjects(data json.RawMessage, fields []string) ([]byte, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	var slim []map[string]interface{}
	for _, item := range items {
		var full map[string]interface{}
		if err := json.Unmarshal(item, &full); err != nil {
			continue
		}
		s := make(map[string]interface{}, len(fields))
		for _, f := range fields {
			s[f] = full[f]
		}
		slim = append(slim, s)
	}
	return json.MarshalIndent(slim, "", "  ")
}

func readResource(uri string) (mcpserve.ReadResult, error) {
	switch {
	case uri == "github://repos":
		data, err := ghAPI("/user/repos?sort=updated&per_page=30")
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		out, err := slimObjects(data, []string{"full_name", "description", "language", "stargazers_count", "updated_at", "private", "fork"})
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil

	case uri == "github://notifications":
		data, err := ghAPI("/notifications?per_page=30")
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		out, err := slimObjects(data, []string{"id", "reason", "unread", "updated_at", "subject"})
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil

	case uri == "github://gists":
		data, err := ghAPI("/gists?per_page=30")
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		out, err := slimObjects(data, []string{"id", "description", "public", "created_at", "updated_at", "files"})
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil

	default:
		return readRepoResource(uri)
	}
}

func readRepoResource(uri string) (mcpserve.ReadResult, error) {
	owner, repo := parseRepo(uri)
	if owner == "" {
		return mcpserve.ReadResult{}, fmt.Errorf("invalid URI: %s", uri)
	}

	suffix := repoSuffix(uri, owner, repo)

	switch suffix {
	case "":
		// Repo details
		data, err := ghAPI(fmt.Sprintf("/repos/%s/%s", owner, repo))
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		out, _ := json.MarshalIndent(json.RawMessage(data), "", "  ")
		return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil

	case "readme":
		return readReadme(owner, repo)

	case "issues":
		data, err := ghAPI(fmt.Sprintf("/repos/%s/%s/issues?state=open&per_page=30", owner, repo))
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		out, err := slimObjects(data, []string{"number", "title", "state", "user", "labels", "created_at", "updated_at"})
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil

	case "pulls":
		data, err := ghAPI(fmt.Sprintf("/repos/%s/%s/pulls?state=open&per_page=30", owner, repo))
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		out, err := slimObjects(data, []string{"number", "title", "state", "user", "head", "base", "created_at", "updated_at"})
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil

	case "actions":
		data, err := ghAPI(fmt.Sprintf("/repos/%s/%s/actions/runs?per_page=10", owner, repo))
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		// Extract workflow_runs array and slim it
		var wrapper struct {
			Runs json.RawMessage `json:"workflow_runs"`
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			return mcpserve.ReadResult{}, err
		}
		out, err := slimObjects(wrapper.Runs, []string{"id", "name", "status", "conclusion", "created_at", "html_url", "head_branch"})
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil

	case "releases":
		data, err := ghAPI(fmt.Sprintf("/repos/%s/%s/releases?per_page=10", owner, repo))
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		out, err := slimObjects(data, []string{"tag_name", "name", "draft", "prerelease", "published_at", "html_url"})
		if err != nil {
			return mcpserve.ReadResult{}, err
		}
		return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil

	default:
		return mcpserve.ReadResult{}, fmt.Errorf("unknown resource: %s", uri)
	}
}

func readReadme(owner, repo string) (mcpserve.ReadResult, error) {
	data, err := ghAPI(fmt.Sprintf("/repos/%s/%s/readme", owner, repo))
	if err != nil {
		return mcpserve.ReadResult{}, err
	}
	var readme struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal(data, &readme); err != nil {
		return mcpserve.ReadResult{}, err
	}
	if readme.Encoding == "base64" {
		// GitHub base64 content has newlines — strip them before decoding
		cleaned := strings.ReplaceAll(readme.Content, "\n", "")
		decoded, err := base64.StdEncoding.DecodeString(cleaned)
		if err != nil {
			return mcpserve.ReadResult{}, fmt.Errorf("base64 decode: %w", err)
		}
		return mcpserve.ReadResult{Text: string(decoded), MimeType: "text/plain"}, nil
	}
	return mcpserve.ReadResult{Text: readme.Content, MimeType: "text/plain"}, nil
}

// parseRepo extracts owner and repo from "github://repos/{owner}/{repo}[/...]"
func parseRepo(uri string) (string, string) {
	path := strings.TrimPrefix(uri, "github://repos/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}

// repoSuffix returns the part after "github://repos/{owner}/{repo}/"
func repoSuffix(uri, owner, repo string) string {
	prefix := fmt.Sprintf("github://repos/%s/%s", owner, repo)
	rest := strings.TrimPrefix(uri, prefix)
	return strings.TrimPrefix(rest, "/")
}

func main() {
	// CLI tool dispatch mode: mcpfs-github <tool-name> [--flags]
	if len(os.Args) > 1 {
		var tools []mcptool.ToolDef
		json.Unmarshal(toolSchemas, &tools)
		caller := &mcptool.HTTPCaller{
			URL:        mcpURL(),
			AuthHeader: "Bearer " + token,
		}
		os.Exit(mcptool.Run("mcpfs-github", tools, caller, os.Args[1:]))
	}

	srv := mcpserve.New("mcpfs-github", "0.1.0", readResource)

	srv.AddResource(mcpserve.Resource{
		URI: "github://repos", Name: "repos",
		Description: "User's repositories", MimeType: "application/json",
	})
	srv.AddResource(mcpserve.Resource{
		URI: "github://notifications", Name: "notifications",
		Description: "Unread notifications", MimeType: "application/json",
	})
	srv.AddResource(mcpserve.Resource{
		URI: "github://gists", Name: "gists",
		Description: "User's gists", MimeType: "application/json",
	})

	srv.AddTemplate(mcpserve.Template{
		URITemplate: "github://repos/{owner}/{repo}", Name: "repo",
		Description: "Repository details", MimeType: "application/json",
	})
	srv.AddTemplate(mcpserve.Template{
		URITemplate: "github://repos/{owner}/{repo}/issues", Name: "issues",
		Description: "Open issues", MimeType: "application/json",
	})
	srv.AddTemplate(mcpserve.Template{
		URITemplate: "github://repos/{owner}/{repo}/pulls", Name: "pulls",
		Description: "Open pull requests", MimeType: "application/json",
	})
	srv.AddTemplate(mcpserve.Template{
		URITemplate: "github://repos/{owner}/{repo}/readme", Name: "readme",
		Description: "README content", MimeType: "text/plain",
	})
	srv.AddTemplate(mcpserve.Template{
		URITemplate: "github://repos/{owner}/{repo}/actions", Name: "actions",
		Description: "Recent workflow runs", MimeType: "application/json",
	})
	srv.AddTemplate(mcpserve.Template{
		URITemplate: "github://repos/{owner}/{repo}/releases", Name: "releases",
		Description: "Releases", MimeType: "application/json",
	})

	if err := srv.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "mcpfs-github: %v\n", err)
		os.Exit(1)
	}
}
