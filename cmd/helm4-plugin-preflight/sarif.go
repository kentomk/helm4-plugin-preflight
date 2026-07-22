package main

import (
	"encoding/json"
	"io"

	"github.com/kentomk/helm4-plugin-preflight/internal/analyzer"
)

const sarifSchema = "https://json.schemastore.org/sarif-2.1.0.json"

type sarifLog struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name            string      `json:"name"`
	SemanticVersion string      `json:"semanticVersion"`
	InformationURI  string      `json:"informationUri"`
	Rules           []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string       `json:"id"`
	ShortDescription sarifMessage `json:"shortDescription"`
	Help             sarifMessage `json:"help"`
	DefaultConfig    sarifConfig  `json:"defaultConfiguration"`
}

type sarifConfig struct {
	Level string `json:"level"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
}

var sarifRules = []sarifRule{
	newSARIFRule("H4P001", "Plugin signature verification is disabled", "Use a source with provenance, replace the plugin, or pin Helm safely while the plugin migrates.", "error"),
	newSARIFRule("H4P002", "Executable path is used as a post-renderer", "Replace the executable path with the name of a Helm 4 postrenderer/v1 plugin.", "error"),
	newSARIFRule("H4P003", "Installed plugin metadata uses the legacy schema", "Add the Helm 4 apiVersion and plugin type to plugin.yaml.", "warning"),
	newSARIFRule("H4P004", "Plugin provenance is unknown from installed metadata", "Verify the original plugin archive or source with Helm native plugin verify.", "warning"),
	newSARIFRule("H4P005", "Installed plugin input was not provided", "Supply --helm-plugins to cross-check repository invocations against installed metadata.", "note"),
}

func newSARIFRule(id, description, help, level string) sarifRule {
	return sarifRule{
		ID:               id,
		ShortDescription: sarifMessage{Text: description},
		Help:             sarifMessage{Text: help},
		DefaultConfig:    sarifConfig{Level: level},
	}
}

func writeSARIF(out io.Writer, report analyzer.Report) error {
	results := make([]sarifResult, 0, len(report.Diagnostics))
	for _, diagnostic := range report.Diagnostics {
		results = append(results, sarifResult{
			RuleID:  diagnostic.RuleID,
			Level:   diagnostic.Severity,
			Message: sarifMessage{Text: diagnostic.Message + " Remediation: " + diagnostic.Remediation},
			Locations: []sarifLocation{{PhysicalLocation: sarifPhysicalLocation{
				ArtifactLocation: sarifArtifactLocation{URI: diagnostic.Path, URIBaseID: "%SRCROOT%"},
				Region:           sarifRegion{StartLine: diagnostic.Line, StartColumn: diagnostic.Column},
			}}},
		})
	}
	log := sarifLog{
		Version: "2.1.0",
		Schema:  sarifSchema,
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:            "helm4-plugin-preflight",
				SemanticVersion: report.ToolVersion,
				InformationURI:  "https://github.com/kentomk/helm4-plugin-preflight",
				Rules:           sarifRules,
			}},
			Results: results,
		}},
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(log)
}
