package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestSARIFOutputMapsRulesLevelsAndLocations(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{"check", "--root", "../../testdata/unsigned-bypass", "--format", "sarif"}, &stdout, &stderr)
	if exit != 1 {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr.String())
	}
	var log sarifLog
	if err := json.Unmarshal(stdout.Bytes(), &log); err != nil {
		t.Fatalf("decode SARIF: %v\n%s", err, stdout.String())
	}
	if log.Version != "2.1.0" || log.Schema != sarifSchema || len(log.Runs) != 1 {
		t.Fatalf("invalid SARIF envelope: %+v", log)
	}
	if len(log.Runs[0].Tool.Driver.Rules) != 5 {
		t.Fatalf("rules = %d, want 5", len(log.Runs[0].Tool.Driver.Rules))
	}
	results := log.Runs[0].Results
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2: %+v", len(results), results)
	}
	if results[0].RuleID != "H4P005" || results[0].Level != "note" || results[1].RuleID != "H4P001" || results[1].Level != "error" {
		t.Fatalf("unexpected result mapping: %+v", results)
	}
	location := results[1].Locations[0].PhysicalLocation
	if location.ArtifactLocation.URI != ".github/workflows/deploy.yml" || location.ArtifactLocation.URIBaseID != "%SRCROOT%" || location.Region.StartLine < 1 || location.Region.StartColumn < 1 {
		t.Fatalf("invalid location: %+v", location)
	}
}

func TestSARIFNoFindingsHasEmptyResults(t *testing.T) {
	t.Parallel()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{"check", "--root", t.TempDir(), "--format", "sarif"}, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit = %d, stderr = %s", exit, stderr.String())
	}
	var log sarifLog
	if err := json.Unmarshal(stdout.Bytes(), &log); err != nil {
		t.Fatal(err)
	}
	if len(log.Runs) != 1 || log.Runs[0].Results == nil || len(log.Runs[0].Results) != 0 {
		t.Fatalf("expected one run with an empty results array: %+v", log)
	}
}
