package config

import (
	"os"
	"testing"
)

func TestParseConfig(t *testing.T) {
	data := []byte(`{
		"posthog": {
			"type": "http",
			"url": "https://mcp.posthog.com/mcp",
			"headers": {"Authorization": "Bearer ${POSTHOG_KEY}"}
		},
		"github": {
			"command": "npx",
			"args": ["-y", "@modelcontextprotocol/server-github"],
			"env": {"GITHUB_TOKEN": "${GITHUB_TOKEN}"}
		}
	}`)

	os.Setenv("POSTHOG_KEY", "phx_test123")
	os.Setenv("GITHUB_TOKEN", "ghp_test456")
	defer os.Unsetenv("POSTHOG_KEY")
	defer os.Unsetenv("GITHUB_TOKEN")

	cfg, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg) != 2 {
		t.Fatalf("got %d servers, want 2", len(cfg))
	}

	ph := cfg["posthog"]
	if ph.Type != "http" {
		t.Errorf("posthog type = %q, want http", ph.Type)
	}
	if ph.URL != "https://mcp.posthog.com/mcp" {
		t.Errorf("posthog url = %q", ph.URL)
	}
	if ph.Headers["Authorization"] != "Bearer phx_test123" {
		t.Errorf("posthog auth = %q (env not interpolated)", ph.Headers["Authorization"])
	}

	gh := cfg["github"]
	if gh.Command != "npx" {
		t.Errorf("github command = %q", gh.Command)
	}
	if gh.Env["GITHUB_TOKEN"] != "ghp_test456" {
		t.Errorf("github env = %q (env not interpolated)", gh.Env["GITHUB_TOKEN"])
	}
}

func TestInterpolateEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "hello")
	defer os.Unsetenv("TEST_VAR")

	cases := []struct{ input, want string }{
		{"Bearer ${TEST_VAR}", "Bearer hello"},
		{"no-vars", "no-vars"},
		{"${MISSING_VAR}", ""},
		{"${TEST_VAR} and ${TEST_VAR}", "hello and hello"},
	}
	for _, tc := range cases {
		got := interpolateEnv(tc.input)
		if got != tc.want {
			t.Errorf("interpolateEnv(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
