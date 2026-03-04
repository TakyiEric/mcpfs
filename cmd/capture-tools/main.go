// capture-tools connects to an MCP server and saves its tool definitions.
// Supports HTTP (SSE/JSON) and stdio transports.
//
// Usage:
//
//	# HTTP transport
//	go run ./cmd/capture-tools -url https://mcp.posthog.com/mcp -auth "Bearer $KEY" -out tools.json
//
//	# Stdio transport
//	go run ./cmd/capture-tools -cmd npx -args @modelcontextprotocol/server-linear -out tools.json
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/airshelf/mcpfs/pkg/mcpclient"
)

var sessionID string

func main() {
	url := flag.String("url", "", "MCP server URL (HTTP transport)")
	auth := flag.String("auth", "", "Authorization header value (HTTP)")
	cmd := flag.String("cmd", "", "Command to launch (stdio transport)")
	cmdArgs := flag.String("args", "", "Space-separated args for -cmd")
	out := flag.String("out", "tools.json", "Output file path")
	flag.Parse()

	if *url == "" && *cmd == "" {
		fmt.Fprintln(os.Stderr, "capture-tools: -url or -cmd required")
		os.Exit(1)
	}

	var toolsJSON json.RawMessage
	var err error

	if *cmd != "" {
		toolsJSON, err = captureStdio(*cmd, *cmdArgs)
	} else {
		toolsJSON, err = captureHTTP(*url, *auth)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "capture-tools: %v\n", err)
		os.Exit(1)
	}

	// Pretty-print and write.
	var tools interface{}
	json.Unmarshal(toolsJSON, &tools)
	pretty, _ := json.MarshalIndent(tools, "", "  ")

	if err := os.WriteFile(*out, pretty, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "capture-tools: write %s: %v\n", *out, err)
		os.Exit(1)
	}

	var list []interface{}
	json.Unmarshal(toolsJSON, &list)
	fmt.Fprintf(os.Stderr, "capture-tools: wrote %d tools to %s\n", len(list), *out)
}

func captureStdio(command, argsStr string) (json.RawMessage, error) {
	var args []string
	if argsStr != "" {
		args = strings.Fields(argsStr)
	}

	client, err := mcpclient.New(command, args)
	if err != nil {
		return nil, fmt.Errorf("launch %s: %w", command, err)
	}
	defer client.Close()

	return client.ListTools()
}

func captureHTTP(url, auth string) (json.RawMessage, error) {
	// Initialize.
	_, err := rpc(url, auth, 1, "initialize", map[string]interface{}{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]string{"name": "capture-tools", "version": "0.1.0"},
	})
	if err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	// List tools.
	result, err := rpc(url, auth, 2, "tools/list", map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("tools/list: %w", err)
	}

	var toolsResult struct {
		Tools json.RawMessage `json:"tools"`
	}
	if err := json.Unmarshal(result, &toolsResult); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return toolsResult.Tools, nil
}

// rpc sends a JSON-RPC request and handles both SSE and plain JSON responses.
func rpc(url, auth string, id int, method string, params interface{}) (json.RawMessage, error) {
	body := map[string]interface{}{
		"jsonrpc": "2.0", "id": id,
		"method": method,
		"params": params,
	}
	data, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Capture session ID from response.
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		sessionID = sid
	}

	if resp.StatusCode >= 400 {
		scanner := bufio.NewScanner(resp.Body)
		var msg string
		if scanner.Scan() {
			msg = scanner.Text()
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, msg)
	}

	ct := resp.Header.Get("Content-Type")

	// SSE response: parse "data:" lines.
	if strings.Contains(ct, "text/event-stream") {
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // 1MB buffer for large tool lists
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			jsonData := strings.TrimPrefix(line, "data: ")
			var rpcResp struct {
				Result json.RawMessage `json:"result"`
				Error  *struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(jsonData), &rpcResp); err != nil {
				continue
			}
			if rpcResp.Error != nil {
				return nil, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
			}
			return rpcResp.Result, nil
		}
		return nil, fmt.Errorf("no data in SSE response")
	}

	// Plain JSON response.
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(buf.Bytes(), &rpcResp); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	return rpcResp.Result, nil
}
