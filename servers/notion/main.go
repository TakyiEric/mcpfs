// mcpfs-notion: Notion MCP resource server for mcpfs.
// Uses mcpserve framework. Speaks MCP JSON-RPC over stdio.
//
// Resources:
//   notion://pages                          - recently edited pages
//   notion://pages/{id}                     - page content (blocks)
//   notion://databases                      - all databases
//   notion://databases/{id}                 - database rows (query)
//   notion://search                         - search all content
//   notion://users                          - workspace users
//
// Auth: NOTION_API_KEY env var (internal integration token, ntn_...).
//       Create at https://www.notion.so/my-integrations
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/airshelf/mcpfs/pkg/mcpserve"
	"github.com/airshelf/mcpfs/pkg/mcptool"
)

//go:embed tools.json
var toolSchemas []byte

var apiKey string

const notionVersion = "2022-06-28"

func notionRequest(method, path string, body interface{}) (json.RawMessage, error) {
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}

	req, _ := http.NewRequest(method, "https://api.notion.com/v1"+path, bodyReader)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Notion-Version", notionVersion)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("notion %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}
	return respBody, nil
}

func mcpURL() string {
	if u := os.Getenv("NOTION_MCP_URL"); u != "" {
		return u
	}
	return "https://mcp.notion.com/mcp"
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func readResource(uri string) (mcpserve.ReadResult, error) {
	switch {
	case uri == "notion://pages":
		return readPages()
	case uri == "notion://databases":
		return readDatabases()
	case uri == "notion://search":
		return readSearch()
	case uri == "notion://users":
		return readUsers()

	case strings.HasPrefix(uri, "notion://pages/"):
		id := strings.TrimPrefix(uri, "notion://pages/")
		return readPageBlocks(id)
	case strings.HasPrefix(uri, "notion://databases/"):
		id := strings.TrimPrefix(uri, "notion://databases/")
		return readDatabaseRows(id)

	default:
		return mcpserve.ReadResult{}, fmt.Errorf("unknown resource: %s", uri)
	}
}

func readPages() (mcpserve.ReadResult, error) {
	raw, err := notionRequest("POST", "/search", map[string]interface{}{
		"filter":    map[string]string{"property": "object", "value": "page"},
		"sort":      map[string]string{"direction": "descending", "timestamp": "last_edited_time"},
		"page_size": 50,
	})
	if err != nil {
		return mcpserve.ReadResult{}, err
	}

	results, _ := extractResults(raw)
	// Slim: just id, title, last_edited, url
	var pages []map[string]interface{}
	var items []json.RawMessage
	json.Unmarshal(results, &items)
	for _, item := range items {
		var page struct {
			ID             string                 `json:"id"`
			URL            string                 `json:"url"`
			LastEditedTime string                 `json:"last_edited_time"`
			Properties     map[string]interface{} `json:"properties"`
		}
		json.Unmarshal(item, &page)
		title := extractTitle(page.Properties)
		pages = append(pages, map[string]interface{}{
			"id": page.ID, "title": title, "url": page.URL,
			"last_edited": page.LastEditedTime,
		})
	}

	out, _ := json.MarshalIndent(pages, "", "  ")
	return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil
}

func readDatabases() (mcpserve.ReadResult, error) {
	raw, err := notionRequest("POST", "/search", map[string]interface{}{
		"filter":    map[string]string{"property": "object", "value": "database"},
		"page_size": 50,
	})
	if err != nil {
		return mcpserve.ReadResult{}, err
	}
	results, _ := extractResults(raw)

	var dbs []map[string]interface{}
	var items []json.RawMessage
	json.Unmarshal(results, &items)
	for _, item := range items {
		var db struct {
			ID    string                   `json:"id"`
			URL   string                   `json:"url"`
			Title []map[string]interface{} `json:"title"`
		}
		json.Unmarshal(item, &db)
		title := ""
		if len(db.Title) > 0 {
			if t, ok := db.Title[0]["plain_text"].(string); ok {
				title = t
			}
		}
		dbs = append(dbs, map[string]interface{}{
			"id": db.ID, "title": title, "url": db.URL,
		})
	}

	out, _ := json.MarshalIndent(dbs, "", "  ")
	return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil
}

func readPageBlocks(id string) (mcpserve.ReadResult, error) {
	raw, err := notionRequest("GET", "/blocks/"+id+"/children?page_size=100", nil)
	if err != nil {
		return mcpserve.ReadResult{}, err
	}
	results, _ := extractResults(raw)

	// Convert blocks to readable text.
	var blocks []json.RawMessage
	json.Unmarshal(results, &blocks)

	var text strings.Builder
	for _, block := range blocks {
		var b struct {
			Type string `json:"type"`
		}
		json.Unmarshal(block, &b)

		// Extract rich text from block.
		var generic map[string]json.RawMessage
		json.Unmarshal(block, &generic)
		if typeData, ok := generic[b.Type]; ok {
			var content struct {
				RichText []struct {
					PlainText string `json:"plain_text"`
				} `json:"rich_text"`
			}
			json.Unmarshal(typeData, &content)
			for _, rt := range content.RichText {
				text.WriteString(rt.PlainText)
			}
		}
		text.WriteString("\n")
	}

	return mcpserve.ReadResult{Text: text.String(), MimeType: "text/plain"}, nil
}

func readDatabaseRows(id string) (mcpserve.ReadResult, error) {
	raw, err := notionRequest("POST", "/databases/"+id+"/query", map[string]interface{}{
		"page_size": 100,
	})
	if err != nil {
		return mcpserve.ReadResult{}, err
	}
	results, _ := extractResults(raw)

	var v interface{}
	json.Unmarshal(results, &v)
	out, _ := json.MarshalIndent(v, "", "  ")
	return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil
}

func readSearch() (mcpserve.ReadResult, error) {
	raw, err := notionRequest("POST", "/search", map[string]interface{}{
		"page_size": 50,
		"sort":      map[string]string{"direction": "descending", "timestamp": "last_edited_time"},
	})
	if err != nil {
		return mcpserve.ReadResult{}, err
	}
	results, _ := extractResults(raw)

	var v interface{}
	json.Unmarshal(results, &v)
	out, _ := json.MarshalIndent(v, "", "  ")
	return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil
}

func readUsers() (mcpserve.ReadResult, error) {
	raw, err := notionRequest("GET", "/users?page_size=100", nil)
	if err != nil {
		return mcpserve.ReadResult{}, err
	}
	results, _ := extractResults(raw)

	var v interface{}
	json.Unmarshal(results, &v)
	out, _ := json.MarshalIndent(v, "", "  ")
	return mcpserve.ReadResult{Text: string(out), MimeType: "application/json"}, nil
}

func extractResults(raw json.RawMessage) (json.RawMessage, error) {
	var paged struct {
		Results json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal(raw, &paged); err == nil && paged.Results != nil {
		return paged.Results, nil
	}
	return raw, nil
}

func extractTitle(props map[string]interface{}) string {
	for _, v := range props {
		m, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		if m["type"] == "title" {
			if titleArr, ok := m["title"].([]interface{}); ok && len(titleArr) > 0 {
				if t, ok := titleArr[0].(map[string]interface{}); ok {
					if pt, ok := t["plain_text"].(string); ok {
						return pt
					}
				}
			}
		}
	}
	return ""
}

func main() {
	apiKey = os.Getenv("NOTION_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "mcpfs-notion: NOTION_API_KEY env var required")
		os.Exit(1)
	}

	// CLI tool dispatch mode: mcpfs-notion <tool-name> [--flags]
	if len(os.Args) > 1 {
		var tools []mcptool.ToolDef
		json.Unmarshal(toolSchemas, &tools)
		caller := &mcptool.HTTPCaller{
			URL:        mcpURL(),
			AuthHeader: "Bearer " + apiKey,
		}
		os.Exit(mcptool.Run("mcpfs-notion", tools, caller, os.Args[1:]))
	}

	srv := mcpserve.New("mcpfs-notion", "0.1.0", readResource)

	srv.AddResource(mcpserve.Resource{
		URI: "notion://pages", Name: "pages",
		Description: "Recently edited pages", MimeType: "application/json",
	})
	srv.AddResource(mcpserve.Resource{
		URI: "notion://databases", Name: "databases",
		Description: "All databases", MimeType: "application/json",
	})
	srv.AddResource(mcpserve.Resource{
		URI: "notion://search", Name: "search",
		Description: "All content (recent first)", MimeType: "application/json",
	})
	srv.AddResource(mcpserve.Resource{
		URI: "notion://users", Name: "users",
		Description: "Workspace users", MimeType: "application/json",
	})

	srv.AddTemplate(mcpserve.Template{
		URITemplate: "notion://pages/{id}", Name: "page",
		Description: "Page content as text", MimeType: "text/plain",
	})
	srv.AddTemplate(mcpserve.Template{
		URITemplate: "notion://databases/{id}", Name: "database",
		Description: "Database rows (query)", MimeType: "application/json",
	})

	if err := srv.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "mcpfs-notion: %v\n", err)
		os.Exit(1)
	}
}
