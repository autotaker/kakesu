package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadWireConfig(t *testing.T) {
	t.Parallel()
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	config, err := LoadWireConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if config.Version != 1 || config.SchemaCatalogRoot != filepath.Join(root, "schemas", "draft-v0") {
		t.Fatalf("unexpected config: %#v", config)
	}
	if len(config.Components) != 3 {
		t.Fatalf("expected three components, got %d", len(config.Components))
	}
}

func TestWireConfigRejectsInvalidBoundaries(t *testing.T) {
	t.Parallel()
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "configs", "local", "wire.json"))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][2]string{
		"version":            {`"version": 1`, `"version": 2`},
		"absolute catalog":   {`"schema_catalog_root": "schemas/draft-v0"`, `"schema_catalog_root": "/schemas"`},
		"parent endpoint":    {`run/core-runtime.sock`, `../core.sock`},
		"duplicate endpoint": {`run/core-runtime.sock`, `run/memory-service.sock`},
		"unknown component":  {`"core-runtime"`, `"extra-service": { "uds_endpoint": "run/extra.sock" }, "core-runtime"`},
	}
	for name, replacement := range cases {
		bad := strings.Replace(string(raw), replacement[0], replacement[1], 1)
		if _, err := ParseWireConfig([]byte(bad), root); err == nil {
			t.Errorf("%s config must be rejected", name)
		}
	}
}
