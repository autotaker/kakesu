package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var componentNames = []string{"core-runtime", "memory-service", "governance-service"}

type WireConfig struct {
	Version           int
	SchemaCatalogRoot string
	Components        map[string]ComponentConfig
}

type ComponentConfig struct {
	UDSEndpoint string
}

// LoadWireConfig reads the single local wire configuration relative to projectRoot.
func LoadWireConfig(projectRoot string) (*WireConfig, error) {
	raw, err := os.ReadFile(filepath.Join(projectRoot, "configs", "local", "wire.json"))
	if err != nil {
		return nil, err
	}
	return ParseWireConfig(raw, projectRoot)
}

// ParseWireConfig strictly validates and resolves project-root-relative paths.
func ParseWireConfig(data []byte, projectRoot string) (*WireConfig, error) {
	var wire struct {
		Version           int    `json:"version"`
		SchemaCatalogRoot string `json:"schema_catalog_root"`
		Components        map[string]struct {
			UDSEndpoint string `json:"uds_endpoint"`
		} `json:"components"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return nil, fmt.Errorf("decode wire config: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("wire config must contain one JSON value")
	}
	if wire.Version != 1 {
		return nil, errors.New("wire config version must be 1")
	}
	if !safeRelative(wire.SchemaCatalogRoot) {
		return nil, errors.New("schema_catalog_root must be a safe relative path")
	}
	if len(wire.Components) != len(componentNames) {
		return nil, errors.New("wire config must define exactly three components")
	}
	config := &WireConfig{
		Version:           wire.Version,
		SchemaCatalogRoot: filepath.Join(projectRoot, filepath.FromSlash(wire.SchemaCatalogRoot)),
		Components:        make(map[string]ComponentConfig, len(componentNames)),
	}
	seen := make(map[string]bool, len(componentNames))
	for _, name := range componentNames {
		component, ok := wire.Components[name]
		if !ok {
			return nil, fmt.Errorf("required component %q is missing", name)
		}
		if !safeRelative(component.UDSEndpoint) {
			return nil, fmt.Errorf("component %q has an unsafe UDS endpoint", name)
		}
		clean := filepath.Clean(filepath.FromSlash(component.UDSEndpoint))
		if seen[clean] {
			return nil, errors.New("UDS endpoints must be unique")
		}
		seen[clean] = true
		config.Components[name] = ComponentConfig{UDSEndpoint: filepath.Join(projectRoot, clean)}
	}
	return config, nil
}

func safeRelative(value string) bool {
	if value == "" || filepath.IsAbs(value) {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(value), "/") {
		if part == ".." {
			return false
		}
	}
	clean := filepath.Clean(filepath.FromSlash(value))
	return clean != "." && !filepath.IsAbs(clean)
}
