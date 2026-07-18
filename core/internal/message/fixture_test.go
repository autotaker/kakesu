package message

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSharedEnvelopeFixtures(t *testing.T) {
	t.Parallel()
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	catalog := filepath.Join(root, "schemas", "draft-v0")
	cases := []struct {
		name  string
		valid bool
	}{
		{"valid.json", true},
		{"invalid-same-route.json", false},
		{"invalid-non-utc-timestamp.json", false},
		{"invalid-schema-reference-digest.json", false},
		{"invalid-non-object-payload.json", false},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, "fixtures", "wire", "envelope", testCase.name))
			if err != nil {
				t.Fatal(err)
			}
			_, err = DecodeAndValidate(raw, catalog)
			if (err == nil) != testCase.valid {
				t.Fatalf("valid=%v, got error %v", testCase.valid, err)
			}
		})
	}
}

func TestEnvelopeStrictFieldsAndExplicitNull(t *testing.T) {
	t.Parallel()
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "fixtures", "wire", "envelope", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	catalog := filepath.Join(root, "schemas", "draft-v0")
	missing := bytes.Replace(raw, []byte("  \"causation_message_id\": null,\n"), nil, 1)
	if _, err := DecodeAndValidate(missing, catalog); err == nil {
		t.Fatal("missing causation_message_id must be rejected")
	}
	unknown := bytes.Replace(raw, []byte("{\n"), []byte("{\n  \"unknown\": true,\n"), 1)
	if _, err := DecodeAndValidate(unknown, catalog); err == nil {
		t.Fatal("unknown envelope field must be rejected")
	}
}

func TestEnvelopeRejectsRawSchemaMutation(t *testing.T) {
	t.Parallel()
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "fixtures", "wire", "envelope", "valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	schema, err := os.ReadFile(filepath.Join(root, "schemas", "draft-v0", "common", "error.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	catalog := t.TempDir()
	if err := os.WriteFile(filepath.Join(catalog, "error.schema.json"), append(schema, ' '), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeAndValidate(raw, catalog); err == nil {
		t.Fatal("one-byte schema mutation must cause a digest mismatch")
	}
}
