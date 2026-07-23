package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopLevelHelpIsDiscoverable(t *testing.T) {
	t.Parallel()
	for _, argument := range []string{"help", "-h", "--help"} {
		argument := argument
		t.Run(argument, func(t *testing.T) {
			t.Parallel()
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exit := run([]string{argument}, &stdout, &stderr)
			if exit != 0 {
				t.Fatalf("exit = %d, stderr = %s", exit, stderr.String())
			}
			for _, expected := range []string{"Usage:", "helm4-plugin-preflight check", "helm4-plugin-preflight version", "check --help"} {
				if !strings.Contains(stdout.String(), expected) {
					t.Fatalf("help is missing %q: %s", expected, stdout.String())
				}
			}
			if stderr.Len() != 0 {
				t.Fatalf("unexpected stderr: %s", stderr.String())
			}
		})
	}
}

func TestNoteOnlyReportExitsZero(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{"check", "--root", "../../testdata/dynamic-unknown"}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr.String())
	}
	if !strings.Contains(stdout.String(), "note H4P005") {
		t.Fatalf("missing H4P005 output: %s", stdout.String())
	}
}

func TestRepeatableShellFileAndEscapeExitCodes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "first.sh"), []byte("helm plugin install demo --verify=false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "second.sh"), []byte("helm template demo . --post-renderer plugin/v1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{"check", "--root", root, "--shell-file", "first.sh", "--shell-file", "second.sh"}, &stdout, &stderr)
	if exit != 1 {
		t.Fatalf("exit = %d, stdout = %s, stderr = %s", exit, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "H4P001") || !strings.Contains(stdout.String(), "H4P005") {
		t.Fatalf("missing shell diagnostics: %s", stdout.String())
	}

	outside := filepath.Join(t.TempDir(), "outside.sh")
	if err := os.WriteFile(outside, []byte("SENSITIVE_OUTSIDE_CONTENT\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	exit = run([]string{"check", "--root", root, "--shell-file", outside}, &stdout, &stderr)
	if exit != 2 {
		t.Fatalf("escape exit = %d, stderr = %s", exit, stderr.String())
	}
	if strings.Contains(stdout.String()+stderr.String(), "SENSITIVE_OUTSIDE_CONTENT") {
		t.Fatal("outside content leaked")
	}
}

func TestMalformedYAMLExitsTwoWithoutDiagnostics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{name: "workflow", args: []string{"check", "--root", "../../testdata/invalid-yaml"}},
		{name: "plugin", args: []string{"check", "--root", "../../testdata/safe-post-renderer", "--helm-plugins", "../../testdata/invalid-plugin/plugins"}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			exit := run(test.args, &stdout, &stderr)
			if exit != 2 {
				t.Fatalf("exit = %d, stdout = %s, stderr = %s", exit, stdout.String(), stderr.String())
			}
			if stdout.Len() != 0 || !strings.Contains(stderr.String(), "invalid YAML") {
				t.Fatalf("unexpected output: stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			for _, content := range []string{"secret", "indentation", "unterminated"} {
				if strings.Contains(stderr.String(), content) {
					t.Fatalf("input content leaked in stderr: %s", stderr.String())
				}
			}
		})
	}
}
