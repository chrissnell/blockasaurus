// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

const apiSpecGolden = "testdata/api_spec_contract.golden"

// specFiles are the OpenAPI documents that define the REST surface. The first
// is an upstream file carrying fork edits — the kind most at risk of being
// reverted wholesale by a merge. The second is Blockasaurus-only.
var specFiles = []string{
	"../docs/api/openapi.yaml",
	"../docs/api/openapi-config.yaml",
}

// httpMethods are the keys under a path item that denote an operation;
// everything else there (parameters, summary, ...) is not one.
var httpMethods = map[string]bool{
	"get": true, "put": true, "post": true, "delete": true,
	"options": true, "head": true, "patch": true, "trace": true,
}

// TestAPISpecContract locks the shape of the REST API as declared in the
// OpenAPI specs: every operation, and every schema those operations exchange,
// down to property names and required fields.
//
// TestAPIContract covers routing — which URLs exist and what guards them.
// This covers payloads. Together they are what "don't change the API
// contract" means in practice, because the Svelte UI in web/ui depends on
// both and a merge can break either one independently.
//
// Deliberately not a checksum of the files: the spec carries prose
// (descriptions, branding) that will legitimately change during an upstream
// merge, and a guard that cries wolf on a reworded description gets
// regenerated without being read.
func TestAPISpecContract(t *testing.T) {
	var lines []string

	for _, path := range specFiles {
		spec, err := loadSpec(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}

		lines = append(lines, "### "+strings.TrimPrefix(path, "../"))
		lines = append(lines, specOperations(spec)...)
		lines = append(lines, specSchemas(spec)...)
		lines = append(lines, "")
	}

	assertGolden(t, apiSpecGolden, strings.Join(lines, "\n"),
		"OpenAPI contract changed",
		"go test ./server -run TestAPISpecContract -update-api-contract")
}

type openAPISpec struct {
	Paths      map[string]map[string]specOperation `yaml:"paths"`
	Components struct {
		Schemas map[string]specSchema `yaml:"schemas"`
	} `yaml:"components"`
}

type specOperation struct {
	OperationID string `yaml:"operationId"`
}

type specSchema struct {
	Type       string                        `yaml:"type"`
	Required   []string                      `yaml:"required"`
	Properties map[string]specSchemaProperty `yaml:"properties"`
}

type specSchemaProperty struct {
	Type   string `yaml:"type"`
	Format string `yaml:"format"`
}

func loadSpec(path string) (*openAPISpec, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // fixed in-repo spec path
	if err != nil {
		return nil, err
	}

	var spec openAPISpec
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		return nil, err
	}

	return &spec, nil
}

func specOperations(spec *openAPISpec) []string {
	var lines []string

	for path, item := range spec.Paths {
		for method, op := range item {
			if !httpMethods[strings.ToLower(method)] {
				continue
			}

			id := op.OperationID
			if id == "" {
				id = "(no operationId)"
			}

			lines = append(lines, fmt.Sprintf("op   %-7s %-45s %s",
				strings.ToUpper(method), path, id))
		}
	}

	sort.Strings(lines)

	return lines
}

func specSchemas(spec *openAPISpec) []string {
	var lines []string

	for name, schema := range spec.Components.Schemas {
		required := append([]string(nil), schema.Required...)
		sort.Strings(required)

		props := make([]string, 0, len(schema.Properties))

		for prop, def := range schema.Properties {
			desc := def.Type
			if def.Format != "" {
				desc += "/" + def.Format
			}

			if desc == "" {
				desc = "?"
			}

			props = append(props, prop+":"+desc)
		}

		sort.Strings(props)

		lines = append(lines, fmt.Sprintf("type %-30s %s required=[%s] props=[%s]",
			name, schema.Type, strings.Join(required, " "), strings.Join(props, " ")))
	}

	sort.Strings(lines)

	return lines
}
