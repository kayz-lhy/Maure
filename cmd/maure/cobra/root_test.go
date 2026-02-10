package maurecobra

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(rawArgs []string, stderr *bytes.Buffer) error {
	var stdout bytes.Buffer
	cmd := NewRootCommand(Config{RawArgs: rawArgs, Stdout: &stdout, Stderr: stderr})
	cmd.SetArgs(NormalizeLegacyArgs(rawArgs, stderr))
	return cmd.Execute()
}

func TestParseLogDeprecatedFormatCompatible(t *testing.T) {
	indexDir := filepath.Join(t.TempDir(), "index")
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		t.Fatalf("mkdir index failed: %v", err)
	}

	logFile := filepath.Join(t.TempDir(), "app.log")
	content := "{\"timestamp\":\"2026-02-10T09:00:00Z\",\"level\":\"error\",\"message\":\"request failed\"}\n"
	if err := os.WriteFile(logFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write log file failed: %v", err)
	}

	var stderr bytes.Buffer
	if err := runCLI([]string{"init", indexDir}, &stderr); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if err := runCLI([]string{"--index", indexDir, "parse-log", logFile, "--format=json"}, &stderr); err != nil {
		t.Fatalf("parse-log failed: %v", err)
	}
	if !strings.Contains(stderr.String(), "--format is deprecated") {
		t.Fatalf("expected deprecated flag warning, got: %s", stderr.String())
	}
}

func TestSearchLegacyOrderWarning(t *testing.T) {
	indexDir := filepath.Join(t.TempDir(), "index")
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		t.Fatalf("mkdir index failed: %v", err)
	}

	logFile := filepath.Join(t.TempDir(), "app.log")
	content := "{\"timestamp\":\"2026-02-10T09:00:00Z\",\"level\":\"error\",\"message\":\"request failed\"}\n"
	if err := os.WriteFile(logFile, []byte(content), 0o644); err != nil {
		t.Fatalf("write log file failed: %v", err)
	}

	var stderr bytes.Buffer
	if err := runCLI([]string{"init", indexDir}, &stderr); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if err := runCLI([]string{"--index", indexDir, "parse-log", "--log-format=json", logFile}, &stderr); err != nil {
		t.Fatalf("parse-log failed: %v", err)
	}
	stderr.Reset()

	if err := runCLI([]string{"--index", indexDir, "search", "request", "--group=level"}, &stderr); err != nil {
		t.Fatalf("legacy search failed: %v", err)
	}
	if !strings.Contains(stderr.String(), "legacy argument order is deprecated") {
		t.Fatalf("expected legacy order warning, got: %s", stderr.String())
	}

	stderr.Reset()
	if err := runCLI([]string{"--index", indexDir, "search", "--group=level", "request"}, &stderr); err != nil {
		t.Fatalf("recommended search failed: %v", err)
	}
	if strings.Contains(stderr.String(), "legacy argument order is deprecated") {
		t.Fatalf("did not expect legacy warning for recommended order")
	}
}

func TestHelpOutputsUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewRootCommand(Config{
		RawArgs: []string{"help"},
		Stdout:  &stdout,
		Stderr:  &stderr,
	})
	cmd.SetArgs([]string{"help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("help execute failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("expected help output to contain Usage, got: %q", stdout.String())
	}
}
