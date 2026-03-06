// Package config parses mcpfs server configuration files.
package config

import (
	"encoding/json"
	"os"
	"strings"
)

// ServerConfig describes how to connect to an MCP server.
type ServerConfig struct {
	Type    string            `json:"type"`    // "http" or "" (stdio)
	URL     string            `json:"url"`     // HTTP endpoint
	Headers map[string]string `json:"headers"` // HTTP headers
	Command string            `json:"command"` // stdio command
	Args    []string          `json:"args"`    // stdio args
	Env     map[string]string `json:"env"`     // env vars for stdio
}

// Parse reads a servers.json config and interpolates env vars.
func Parse(data []byte) (map[string]*ServerConfig, error) {
	var raw map[string]*ServerConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	for _, cfg := range raw {
		cfg.URL = interpolateEnv(cfg.URL)
		for k, v := range cfg.Headers {
			cfg.Headers[k] = interpolateEnv(v)
		}
		for k, v := range cfg.Env {
			cfg.Env[k] = interpolateEnv(v)
		}
	}
	return raw, nil
}

// Load reads and parses a config file.
func Load(path string) (map[string]*ServerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

func interpolateEnv(s string) string {
	for {
		start := strings.Index(s, "${")
		if start < 0 {
			return s
		}
		end := strings.Index(s[start:], "}")
		if end < 0 {
			return s
		}
		varName := s[start+2 : start+end]
		s = s[:start] + os.Getenv(varName) + s[start+end+1:]
	}
}
