package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCapabilitySchemasAreValidJSON(t *testing.T) {
	for _, name := range []string{"capability-proposal.schema.json", "capability-descriptor.schema.json"} {
		path := filepath.Join(pluginRoot(), "schemas", name)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		var schema map[string]any
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if schema["$schema"] == nil || schema["title"] == nil {
			t.Errorf("%s is missing $schema/title", name)
		}
	}
}

func TestCapabilityCatalogPassesAudit(t *testing.T) {
	items, err := loadCapabilities()
	if err != nil {
		t.Fatalf("load capabilities: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("capability catalog is empty")
	}
	seen := map[string]bool{}
	for _, item := range items {
		if seen[item.ID] {
			t.Errorf("duplicate capability id: %s", item.ID)
		}
		seen[item.ID] = true
		if item.Schema != capabilitySchema {
			t.Errorf("%s: schema = %q", item.ID, item.Schema)
		}
		result, rpcErr := capabilityAudit(requestInput{CapabilityID: item.ID})
		if rpcErr != nil {
			t.Fatalf("%s: audit failed: %s", item.ID, rpcErr.Message)
		}
		payload := result.(map[string]any)
		if payload["valid"] != true {
			t.Errorf("%s: audit issues = %#v", item.ID, payload["issues"])
		}
	}
}

func TestCapabilityListAndDescribe(t *testing.T) {
	result, rpcErr := capabilityList(requestInput{Status: "ready"})
	if rpcErr != nil {
		t.Fatalf("capability.list: %s", rpcErr.Message)
	}
	payload := result.(map[string]any)
	if payload["total"].(int) == 0 {
		t.Fatal("expected at least one ready capability")
	}

	described, rpcErr := capabilityDescribe(requestInput{CapabilityID: "sankey-flow"})
	if rpcErr != nil {
		t.Fatalf("capability.describe: %s", rpcErr.Message)
	}
	item := described.(map[string]any)["capability"].(videoCapability)
	if item.Template.Composition != "SankeyFlow" {
		t.Fatalf("unexpected composition: %s", item.Template.Composition)
	}
}
