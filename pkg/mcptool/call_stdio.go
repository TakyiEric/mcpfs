package mcptool

import (
	"encoding/json"
	"fmt"

	"github.com/airshelf/mcpfs/pkg/mcpclient"
)

// StdioCaller launches an MCP server subprocess and calls tools via stdio.
type StdioCaller struct {
	Command string   // e.g. "npx"
	Args    []string // e.g. ["@mseep/linear-mcp"]

	client *mcpclient.Client
}

// Call launches the subprocess (if not already running), then calls the tool.
func (c *StdioCaller) Call(toolName string, args map[string]interface{}) (json.RawMessage, error) {
	if c.client == nil {
		client, err := mcpclient.New(c.Command, c.Args)
		if err != nil {
			return nil, fmt.Errorf("launch %s: %w", c.Command, err)
		}
		c.client = client
	}
	return c.client.CallTool(toolName, args)
}

// Close shuts down the subprocess.
func (c *StdioCaller) Close() {
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
}
