package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"himind-plugin/media-workbench/internal/himindjsonrpc"
)

func TestDataIngestCSV(t *testing.T) {
	workspace := t.TempDir()
	path := filepath.Join(workspace, "capitals.csv")
	if err := os.WriteFile(path, []byte("城市,GDP,单位,年份,来源\n合肥,14210,亿元,2025,示例来源\n芜湖,5121,亿元,2025,示例来源\n"), 0644); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"workspace_root": workspace, "path": "capitals.csv"})
	result, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.data.ingest", Params: payload})
	if rpcErr != nil {
		t.Fatalf("data.ingest: %s", rpcErr.Message)
	}
	payloadMap := result.(map[string]any)
	if payloadMap["valid"] != true || payloadMap["count"] != 2 {
		t.Fatalf("unexpected result: %#v", payloadMap)
	}
	items := payloadMap["items"].([]ingestedItem)
	if items[0].Name != "合肥" || items[0].Value != 14210 || items[0].Unit != "亿元" || items[0].Period != "2025" {
		t.Fatalf("unexpected first item: %#v", items[0])
	}
	ranked := payloadMap["ranked_items"].([]ingestedItem)
	if ranked[0].Name != "合肥" {
		t.Fatalf("ranked first = %s", ranked[0].Name)
	}
}

func TestDataIngestRejectsInvalidRow(t *testing.T) {
	workspace := t.TempDir()
	path := filepath.Join(workspace, "bad.csv")
	if err := os.WriteFile(path, []byte("name,value\n合肥,not-a-number\n芜湖,100\n"), 0644); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"workspace_root": workspace, "path": "bad.csv"})
	result, rpcErr := handle(himindjsonrpc.Request{Method: "media.workbench.data.ingest", Params: payload})
	if rpcErr != nil {
		t.Fatalf("data.ingest: %s", rpcErr.Message)
	}
	payloadMap := result.(map[string]any)
	if payloadMap["valid"] != false {
		t.Fatalf("expected invalid result: %#v", payloadMap)
	}
	issues, _ := payloadMap["issues"].([]map[string]any)
	if !hasIssueCode(issues, "data_ingest_value_invalid") {
		t.Fatalf("expected data_ingest_value_invalid, got %#v", issues)
	}
}
