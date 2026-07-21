package analyzer

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCheckGolden(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		root       string
		goldenPath string
	}{
		{name: "unsigned bypass", root: "../../testdata/unsigned-bypass", goldenPath: "testdata/unsigned-bypass.json"},
		{name: "post renderer path", root: "../../testdata/post-renderer-path", goldenPath: "testdata/post-renderer-path.json"},
		{name: "safe post renderer", root: "../../testdata/safe-post-renderer", goldenPath: "testdata/safe-post-renderer.json"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			report, err := Check(test.root, "", nil, "test")
			if err != nil {
				t.Fatal(err)
			}
			report.Root = "FIXTURE"
			actual, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			actual = append(actual, '\n')
			expected, err := os.ReadFile(test.goldenPath)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("golden mismatch\nactual:\n%s\nexpected:\n%s", actual, expected)
			}
		})
	}
}

func TestPluginMetadataInputBoundary(t *testing.T) {
	t.Parallel()
	t.Run("oversized", func(t *testing.T) {
		root := t.TempDir()
		pluginDir := filepath.Join(root, "large")
		if err := os.Mkdir(pluginDir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), bytes.Repeat([]byte("x"), maxPluginBytes+1), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := scanPlugins(root); err == nil {
			t.Fatal("expected oversized metadata to be rejected")
		}
	})
	t.Run("metadata symlink", func(t *testing.T) {
		root := t.TempDir()
		pluginDir := filepath.Join(root, "linked")
		if err := os.Mkdir(pluginDir, 0o700); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, "outside.yaml")
		if err := os.WriteFile(target, []byte("name: outside\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(pluginDir, "plugin.yaml")); err != nil {
			t.Fatal(err)
		}
		if _, _, err := scanPlugins(root); err == nil {
			t.Fatal("expected metadata symlink to be rejected")
		}
	})
}

func TestMalformedYAMLIsRejectedWithoutContent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		root    string
		plugins string
	}{
		{name: "workflow", root: "../../testdata/invalid-yaml"},
		{name: "plugin", root: "../../testdata/safe-post-renderer", plugins: "../../testdata/invalid-plugin/plugins"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := Check(test.root, test.plugins, nil, "test")
			if err == nil {
				t.Fatal("expected malformed YAML to be rejected")
			}
			if !strings.Contains(err.Error(), "invalid YAML") {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, content := range []string{"secret", "unterminated", "indentation"} {
				if strings.Contains(err.Error(), content) {
					t.Fatalf("input content leaked in error: %v", err)
				}
			}
		})
	}
}

func TestYAMLDocumentBoundary(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		data []byte
	}{
		{name: "empty", data: nil},
		{name: "invalid utf8", data: []byte{0xff}},
		{name: "sequence root", data: []byte("- workflow\n")},
		{name: "multiple documents", data: []byte("name: first\n---\nname: hidden\n")},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseYAMLMapping(test.data, "input.yml"); err == nil {
				t.Fatal("expected invalid YAML document boundary to be rejected")
			}
		})
	}
}

func TestCheckMissingWorkflowDirectory(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	report, err := Check(root, "", nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 0 || report.Summary.FilesScanned != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestCheckRejectsFileRoot(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "not-a-root")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(path, "", nil, "test"); err == nil {
		t.Fatal("expected file root to be rejected")
	}
}

func TestInstalledPluginMetadata(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		plugins    string
		wantRules  []string
		wantPlugin string
	}{
		{name: "legacy", plugins: "../../testdata/installed-legacy/plugins", wantRules: []string{"H4P003", "H4P004"}, wantPlugin: "legacy-example"},
		{name: "v1", plugins: "../../testdata/safe-v1-plugin/plugins", wantRules: []string{"H4P004"}, wantPlugin: "v1-example"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			report, err := Check("../../testdata/safe-post-renderer", test.plugins, nil, "test")
			if err != nil {
				t.Fatal(err)
			}
			var rules []string
			for _, diagnostic := range report.Diagnostics {
				rules = append(rules, diagnostic.RuleID)
				if diagnostic.PluginValue != test.wantPlugin {
					t.Fatalf("plugin value = %q, want %q", diagnostic.PluginValue, test.wantPlugin)
				}
			}
			if !reflect.DeepEqual(rules, test.wantRules) {
				t.Fatalf("rules = %#v, want %#v", rules, test.wantRules)
			}
		})
	}
}

func TestMixedRepositoryUsesOnePluginKey(t *testing.T) {
	t.Parallel()
	report, err := Check("../../testdata/mixed-repository", "../../testdata/mixed-repository/plugins", nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 3 {
		t.Fatalf("diagnostics = %d, want 3: %+v", len(report.Diagnostics), report.Diagnostics)
	}
	seenRules := make(map[string]bool)
	for _, diagnostic := range report.Diagnostics {
		seenRules[diagnostic.RuleID] = true
		if diagnostic.PluginValue != "legacy-example" {
			t.Fatalf("plugin key = %q, want legacy-example", diagnostic.PluginValue)
		}
	}
	for _, rule := range []string{"H4P001", "H4P003", "H4P004"} {
		if !seenRules[rule] {
			t.Fatalf("missing %s in %+v", rule, report.Diagnostics)
		}
	}
}

func TestShellTokensPreserveQuotedPath(t *testing.T) {
	t.Parallel()
	actual := shellTokens(`helm template demo . --post-renderer "./scripts/my renderer"`)
	expected := []string{"helm", "template", "demo", ".", "--post-renderer", "./scripts/my renderer"}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("tokens = %#v, want %#v", actual, expected)
	}
}

func TestMissingInstalledInputIsNote(t *testing.T) {
	t.Parallel()
	report, err := Check("../../testdata/dynamic-unknown", "", nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Errors != 0 || report.Summary.Warnings != 0 || report.Summary.Unknowns != 2 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.RuleID != "H4P005" || diagnostic.Severity != "note" {
			t.Fatalf("dynamic input became actionable: %+v", diagnostic)
		}
	}
}

func TestInstalledInputSuppressesMissingInputNote(t *testing.T) {
	t.Parallel()
	report, err := Check("../../testdata/safe-post-renderer", "../../testdata/safe-v1-plugin/plugins", nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.RuleID == "H4P005" {
			t.Fatalf("unexpected missing-input note: %+v", diagnostic)
		}
	}
}

func TestExplicitShellFileAndConfinement(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	script := filepath.Join(root, "deploy.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nhelm plugin install https://example.test/demo --verify=false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Check(root, "", []string{"deploy.sh", "deploy.sh"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.FilesScanned != 1 {
		t.Fatalf("files scanned = %d, want 1", report.Summary.FilesScanned)
	}
	if len(report.Diagnostics) != 2 || report.Diagnostics[0].RuleID != "H4P005" || report.Diagnostics[1].RuleID != "H4P001" {
		t.Fatalf("unexpected diagnostics: %+v", report.Diagnostics)
	}

	outside := filepath.Join(t.TempDir(), "outside.sh")
	if err := os.WriteFile(outside, []byte("helm plugin install secret --verify=false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(root, "", []string{outside}, "test"); err == nil {
		t.Fatal("expected outside shell file to be rejected")
	}
	link := filepath.Join(root, "linked.sh")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(root, "", []string{"linked.sh"}, "test"); err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}
