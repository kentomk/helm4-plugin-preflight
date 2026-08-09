package analyzer

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
)

const (
	maxWorkflowBytes = 2 << 20
	maxPluginBytes   = 256 << 10
)

type Diagnostic struct {
	RuleID      string `json:"ruleId"`
	Severity    string `json:"severity"`
	Path        string `json:"path"`
	Line        int    `json:"line"`
	Column      int    `json:"column"`
	PluginValue string `json:"pluginValue,omitempty"`
	Message     string `json:"message"`
	Remediation string `json:"remediation"`
}

type ScannedFile struct {
	Path string `json:"path"`
}

type Summary struct {
	FilesScanned int `json:"filesScanned"`
	Errors       int `json:"errors"`
	Warnings     int `json:"warnings,omitempty"`
	Unknowns     int `json:"unknowns"`
}

type Report struct {
	SchemaVersion int           `json:"schemaVersion"`
	ToolVersion   string        `json:"toolVersion"`
	Root          string        `json:"root"`
	Scanned       []ScannedFile `json:"scanned"`
	Diagnostics   []Diagnostic  `json:"diagnostics"`
	Summary       Summary       `json:"summary"`
}

func Check(root, helmPlugins string, shellFiles []string, version string) (Report, error) {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return Report{}, fmt.Errorf("resolve root: %w", err)
	}
	info, err := os.Stat(cleanRoot)
	if err != nil {
		return Report{}, fmt.Errorf("read root: %w", err)
	}
	if !info.IsDir() {
		return Report{}, errors.New("root is not a directory")
	}
	cleanRoot, err = filepath.EvalSymlinks(cleanRoot)
	if err != nil {
		return Report{}, fmt.Errorf("resolve root symlinks: %w", err)
	}

	report := Report{
		SchemaVersion: 1,
		ToolVersion:   version,
		Root:          filepath.Clean(root),
		Scanned:       []ScannedFile{},
		Diagnostics:   []Diagnostic{},
	}
	workflowDir := filepath.Join(cleanRoot, ".github", "workflows")
	entries, err := os.ReadDir(workflowDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Report{}, fmt.Errorf("read workflows: %w", err)
	}

	scannedPaths := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yml" && ext != ".yaml" {
			continue
		}
		path := filepath.Join(workflowDir, entry.Name())
		displayPath := filepath.ToSlash(filepath.Join(".github", "workflows", entry.Name()))
		diagnostics, err := scanWorkflow(path, displayPath, helmPlugins == "")
		if err != nil {
			return Report{}, err
		}
		resolvedPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return Report{}, fmt.Errorf("resolve workflow %s: %w", displayPath, err)
		}
		scannedPaths[resolvedPath] = true
		report.Scanned = append(report.Scanned, ScannedFile{Path: displayPath})
		report.Diagnostics = append(report.Diagnostics, diagnostics...)
	}

	for _, shellFile := range shellFiles {
		path, displayPath, err := confinedFile(cleanRoot, shellFile)
		if err != nil {
			return Report{}, err
		}
		if scannedPaths[path] {
			continue
		}
		diagnostics, err := scanShellFile(path, displayPath, helmPlugins == "")
		if err != nil {
			return Report{}, err
		}
		scannedPaths[path] = true
		report.Scanned = append(report.Scanned, ScannedFile{Path: displayPath})
		report.Diagnostics = append(report.Diagnostics, diagnostics...)
	}

	if helmPlugins != "" {
		pluginScanned, pluginDiagnostics, err := scanPlugins(helmPlugins)
		if err != nil {
			return Report{}, err
		}
		report.Scanned = append(report.Scanned, pluginScanned...)
		report.Diagnostics = append(report.Diagnostics, pluginDiagnostics...)
	}

	sort.Slice(report.Scanned, func(i, j int) bool { return report.Scanned[i].Path < report.Scanned[j].Path })
	sort.Slice(report.Diagnostics, func(i, j int) bool {
		a, b := report.Diagnostics[i], report.Diagnostics[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return a.RuleID < b.RuleID
	})
	report.Summary.FilesScanned = len(report.Scanned)
	for _, diagnostic := range report.Diagnostics {
		switch diagnostic.Severity {
		case "error":
			report.Summary.Errors++
		case "warning":
			report.Summary.Warnings++
		}
		if diagnostic.RuleID == "H4P004" || diagnostic.RuleID == "H4P005" {
			report.Summary.Unknowns++
		}
	}
	return report, nil
}

func confinedFile(root, input string) (string, string, error) {
	path := input
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("resolve shell file %q: %w", input, err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", "", fmt.Errorf("resolve shell file %q: %w", input, err)
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return "", "", fmt.Errorf("shell file %q resolves outside repository root", input)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", "", fmt.Errorf("stat shell file %q: %w", input, err)
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("shell file %q is not a regular file", input)
	}
	if info.Size() > maxWorkflowBytes {
		return "", "", fmt.Errorf("shell file %q exceeds %d bytes", input, maxWorkflowBytes)
	}
	return resolved, filepath.ToSlash(relative), nil
}

func scanPlugins(root string) ([]ScannedFile, []Diagnostic, error) {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve Helm plugins directory: %w", err)
	}
	info, err := os.Stat(cleanRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("read Helm plugins directory: %w", err)
	}
	if !info.IsDir() {
		return nil, nil, errors.New("Helm plugins path is not a directory")
	}
	entries, err := os.ReadDir(cleanRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("read Helm plugins directory: %w", err)
	}

	var scanned []ScannedFile
	var diagnostics []Diagnostic
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		metadataPath := filepath.Join(cleanRoot, entry.Name(), "plugin.yaml")
		metadataInfo, err := os.Lstat(metadataPath)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("stat plugin metadata %s: %w", entry.Name(), err)
		}
		if !metadataInfo.Mode().IsRegular() {
			return nil, nil, fmt.Errorf("plugin metadata %s/plugin.yaml is not a regular file", entry.Name())
		}
		if metadataInfo.Size() > maxPluginBytes {
			return nil, nil, fmt.Errorf("plugin metadata %s/plugin.yaml exceeds %d bytes", entry.Name(), maxPluginBytes)
		}
		displayPath := filepath.ToSlash(filepath.Join("plugins", entry.Name(), "plugin.yaml"))
		metadata, err := readPluginMetadata(metadataPath, displayPath)
		if err != nil {
			return nil, nil, err
		}
		scanned = append(scanned, ScannedFile{Path: displayPath})
		pluginName := metadata["name"]
		if pluginName == "" {
			pluginName = entry.Name()
		}
		if metadata["apiVersion"] == "" || metadata["type"] == "" {
			diagnostics = append(diagnostics, Diagnostic{
				RuleID: "H4P003", Severity: "warning", Path: displayPath, Line: 1, Column: 1,
				PluginValue: pluginName,
				Message:     "installed plugin uses legacy metadata without both apiVersion and type",
				Remediation: "migrate the plugin metadata to the Helm 4 v1 schema and declare its plugin type",
			})
		}
		diagnostics = append(diagnostics, Diagnostic{
			RuleID: "H4P004", Severity: "warning", Path: displayPath, Line: 1, Column: 1,
			PluginValue: pluginName,
			Message:     "installed metadata alone cannot establish plugin artifact provenance",
			Remediation: "pass the original plugin archive or source to Helm native plugin verify before migration",
		})
	}
	return scanned, diagnostics, nil
}

func readPluginMetadata(path, displayPath string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", displayPath, err)
	}
	root, err := parseYAMLMapping(data, displayPath)
	if err != nil {
		return nil, err
	}
	metadata := make(map[string]string)
	for index := 0; index < len(root.Content); index += 2 {
		key := root.Content[index]
		value := root.Content[index+1]
		if key.Kind != yaml.ScalarNode {
			continue
		}
		if key.Value != "name" && key.Value != "version" && key.Value != "apiVersion" && key.Value != "type" {
			continue
		}
		if value.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("invalid YAML scalar for %s in %s", key.Value, displayPath)
		}
		metadata[key.Value] = value.Value
	}
	return metadata, nil
}

func parseYAMLMapping(data []byte, displayPath string) (*yaml.Node, error) {
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("invalid YAML in %s", displayPath)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("YAML root in %s must be a mapping", displayPath)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, fmt.Errorf("invalid YAML in %s", displayPath)
		}
		return nil, fmt.Errorf("multiple YAML documents are not supported in %s", displayPath)
	}
	return document.Content[0], nil
}

func scanWorkflow(path, displayPath string, missingPlugins bool) ([]Diagnostic, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", displayPath, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("workflow %s is not a regular file", displayPath)
	}
	if info.Size() > maxWorkflowBytes {
		return nil, fmt.Errorf("workflow %s exceeds %d bytes", displayPath, maxWorkflowBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", displayPath, err)
	}
	if _, err := parseYAMLMapping(data, displayPath); err != nil {
		return nil, err
	}

	var diagnostics []Diagnostic
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		command, column, ok := workflowCommand(line)
		if !ok {
			continue
		}
		diagnostics = append(diagnostics, inspectCommand(command, displayPath, lineNumber, column, missingPlugins)...)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", displayPath, err)
	}
	return diagnostics, nil
}

func scanShellFile(path, displayPath string, missingPlugins bool) ([]Diagnostic, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", displayPath, err)
	}
	defer file.Close()

	var diagnostics []Diagnostic
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		column := strings.Index(line, strings.TrimLeftFunc(line, unicode.IsSpace)) + 1
		diagnostics = append(diagnostics, inspectCommand(trimmed, displayPath, lineNumber, column, missingPlugins)...)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", displayPath, err)
	}
	return diagnostics, nil
}

func workflowCommand(line string) (string, int, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", 0, false
	}
	if index := strings.Index(trimmed, "run:"); index == 0 || (index > 0 && strings.TrimSpace(trimmed[:index]) == "-") {
		value := strings.TrimSpace(trimmed[index+len("run:"):])
		if value == "" || value == "|" || value == ">" || strings.HasPrefix(value, "|") || strings.HasPrefix(value, ">") {
			return "", 0, false
		}
		column := strings.Index(line, value) + 1
		return strings.Trim(value, "\"'"), column, true
	}
	// Indented lines inside a run block are safe to inspect conservatively: only
	// literal lines containing an explicit helm command can produce findings.
	if strings.Contains(trimmed, "helm ") {
		column := strings.Index(line, strings.TrimLeftFunc(line, unicode.IsSpace)) + 1
		return trimmed, column, true
	}
	return "", 0, false
}

func inspectCommand(command, path string, line, baseColumn int, missingPlugins bool) []Diagnostic {
	tokens := shellTokens(command)
	var diagnostics []Diagnostic
	migrationInvocation := false
	invocationValue := ""
	for index := 0; index < len(tokens); index++ {
		if tokens[index] != "helm" {
			continue
		}
		if index+2 < len(tokens) && tokens[index+1] == "plugin" && tokens[index+2] == "install" {
			pluginValue := installedSourceName(tokens[index+3:])
			migrationInvocation = true
			invocationValue = pluginValue
			installTokens := tokens[index+3:]
			for tokenIndex, token := range installTokens {
				verificationDisabled := token == "--verify=false" || token == "--verify=0"
				if token == "--verify" && tokenIndex+1 < len(installTokens) {
					verificationDisabled = installTokens[tokenIndex+1] == "false" || installTokens[tokenIndex+1] == "0"
				}
				if verificationDisabled {
					diagnostics = append(diagnostics, Diagnostic{
						RuleID: "H4P001", Severity: "error", Path: path, Line: line,
						Column: baseColumn + strings.Index(command, token), PluginValue: pluginValue,
						Message:     "plugin installation disables Helm 4 signature verification",
						Remediation: "use a plugin source with provenance, replace the plugin, or pin Helm 3 until a verified migration is available",
					})
				}
			}
		}
		for tokenIndex := index + 1; tokenIndex < len(tokens); tokenIndex++ {
			value := ""
			flagToken := tokens[tokenIndex]
			switch {
			case flagToken == "--post-renderer" && tokenIndex+1 < len(tokens):
				value = tokens[tokenIndex+1]
			case strings.HasPrefix(flagToken, "--post-renderer="):
				value = strings.TrimPrefix(flagToken, "--post-renderer=")
			}
			if value != "" && isExecutablePath(value) {
				diagnostics = append(diagnostics, Diagnostic{
					RuleID: "H4P002", Severity: "error", Path: path, Line: line,
					Column: baseColumn + strings.Index(command, flagToken), PluginValue: value,
					Message:     "Helm 4 requires --post-renderer to name a postrenderer/v1 plugin, not an executable path",
					Remediation: "package the executable as a postrenderer/v1 plugin and pass its plugin name",
				})
			}
			if value != "" {
				migrationInvocation = true
				if invocationValue == "" {
					invocationValue = value
				}
			}
		}
	}
	if missingPlugins && migrationInvocation {
		diagnostics = append(diagnostics, Diagnostic{
			RuleID: "H4P005", Severity: "note", Path: path, Line: line, Column: baseColumn,
			PluginValue: invocationValue,
			Message:     "plugin invocation cannot be cross-checked because installed plugin input was not provided",
			Remediation: "pass --helm-plugins with an explicit local directory, or review this invocation manually",
		})
	}
	return diagnostics
}

func installedSourceName(tokens []string) string {
	for _, token := range tokens {
		if strings.HasPrefix(token, "-") {
			continue
		}
		trimmed := strings.TrimSuffix(strings.TrimRight(token, "/"), ".git")
		if slash := strings.LastIndex(trimmed, "/"); slash >= 0 {
			trimmed = trimmed[slash+1:]
		}
		return trimmed
	}
	return ""
}

func isExecutablePath(value string) bool {
	return strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") || strings.HasPrefix(value, "/")
}

func shellTokens(command string) []string {
	var tokens []string
	var current strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	for _, character := range command {
		if escaped {
			current.WriteRune(character)
			escaped = false
			continue
		}
		if character == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if character == quote {
				quote = 0
			} else {
				current.WriteRune(character)
			}
			continue
		}
		if character == '\'' || character == '"' {
			quote = character
			continue
		}
		if unicode.IsSpace(character) || character == ';' || character == '|' || character == '&' {
			flush()
			continue
		}
		if character == '#' && current.Len() == 0 {
			break
		}
		current.WriteRune(character)
	}
	flush()
	return tokens
}
