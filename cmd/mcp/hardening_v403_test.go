package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestV403MCPAuditDoesNotPersistCredentialValues(t *testing.T) {
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	audit, err := newAuditLogger(auditPath)
	if err != nil { t.Fatal(err) }
	wrapped := wrapToolHandler(toolName, rateLimitRule{}, nil, audit,
		func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultError("blocked"), nil
		})
	_, err = wrapped(context.Background(), scanRequest(map[string]any{
		"target": "https://example.com/?api_key=query-secret-403",
		"approval_token": "approval-secret-403",
		"headers": []any{"Cookie: session=cookie-secret-403", "X-API-Key: header-secret-403"},
		"body": "{\"password\":\"body-secret-403\"}",
		"auth_matrix_json": "{\"admin\":[\"Cookie: session=matrix-secret-403\"]}",
		"unknown_dynamic_key-secret-403": "unknown-value-secret-403",
		"rps": 4,
	}))
	if err != nil { t.Fatal(err) }
	if err := audit.Close(); err != nil { t.Fatal(err) }
	raw, err := os.ReadFile(auditPath)
	if err != nil { t.Fatal(err) }
	for _, secret := range []string{
		"query-secret-403", "approval-secret-403", "cookie-secret-403",
		"header-secret-403", "body-secret-403", "matrix-secret-403",
		"unknown_dynamic_key-secret-403", "unknown-value-secret-403",
	} {
		if strings.Contains(string(raw), secret) { t.Fatalf("audit leaked %q: %s", secret, raw) }
	}
	var entry auditEntry
	if err := json.Unmarshal(raw, &entry); err != nil { t.Fatal(err) }
	args := entry.Arguments.(map[string]any)
	if args["approval_token"] != "[REDACTED]" || args["rps"] != float64(4) {
		t.Fatalf("audit lost safe metadata or token redaction: %#v", args)
	}
}

func TestV403ProbeOriginValidation(t *testing.T) {
	base := "https://api.example.com/base"
	for _, candidate := range []string{
		"https://api.example.com:8443/admin",
		"http://api.example.com/admin",
		"https://evil.example/admin",
		"https://user:pass@api.example.com/admin",
	} {
		t.Run(candidate, func(t *testing.T) {
			if _, err := resolveProbeTarget(base, candidate); err == nil {
				t.Fatalf("unexpectedly accepted probe %q", candidate)
			}
		})
	}
	for _, candidate := range []string{
		"https://api.example.com:443/admin",
		"https://api.example.com/admin",
	} {
		if _, err := resolveProbeTarget(base, candidate); err != nil {
			t.Fatalf("expected equivalent origin %q to pass: %v", candidate, err)
		}
	}
}

func TestV403ExpansionCandidateCannotChangeOrigin(t *testing.T) {
	base := "https://api.example.com/"
	for _, path := range []string{
		"https://other.example/admin",
		"http://api.example.com/admin",
		"https://api.example.com:444/admin",
	} {
		if _, err := resolveProbeTarget(base, path); err == nil {
			t.Fatalf("expansion candidate %q changed origin", path)
		}
	}
	got, err := resolveProbeTarget(base, "@other.example")
	if err != nil { t.Fatal(err) }
	if got != "https://api.example.com/@other.example" {
		t.Fatalf("unsafe URL authority construction: %q", got)
	}
}
