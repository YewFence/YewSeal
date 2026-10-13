package config

import (
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/YewFence/YewSeal/internal/seal"
	toml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

// Alignment anchor between schema/ and Go structs; see schema/config.cue header.
// yewseal.schema.json is exported from config.cue, and mise run schema:check
// guarantees export freshness, so validating the exported JSON Schema is
// equivalent to validating config.cue's structure.
const (
	exampleFile    = "../../schema/example.yewseal.toml"
	jsonSchemaFile = "../../schema/yewseal.schema.json"
)

// TestExampleConfigStrictUnmarshal strict-unmarshals the full-field example,
// ensuring every field in the schema and example has a real home in the Go Config.
func TestExampleConfigStrictUnmarshal(t *testing.T) {
	data, err := os.ReadFile(exampleFile)
	require.NoError(t, err)

	var cfg Config
	decoder := toml.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&cfg))

	// Spot-check key fields to confirm the example maps to the right place,
	// not merely "parses without error".
	require.Equal(t, []string{"target/", "node_modules/", "*.example.toml", "!config/example.toml"}, cfg.Exclude)
	require.Len(t, cfg.Encryption.Files, 2)
	require.Equal(t, "wrangler.toml", cfg.Encryption.Files[0].PlaintextPath)
	require.Equal(t, ".dev.vars", cfg.Encryption.Files[1].PlaintextPath)
	require.Equal(t, "env", cfg.Encryption.Files[1].Format)

	require.Len(t, cfg.Encryption.Groups, 1)
	require.Equal(t, []string{"config/**/*.toml", "secrets/*.yaml"}, cfg.Encryption.Groups[0].Patterns)
	require.Equal(t, []string{"*.dev.vars=env", "**/*.conf=ini"}, cfg.Encryption.Groups[0].FormatRules)
	require.True(t, cfg.Encryption.Groups[0].UnknownAsBinary)
	require.Equal(t, "delivery", cfg.Encryption.Groups[0].PlaintextMode)

	require.NotNil(t, cfg.Recipients.Defaults)
	require.Len(t, *cfg.Recipients.Defaults, 1)
	require.Equal(t, []string{"owner"}, *cfg.Recipients.Defaults)
	require.NotEmpty(t, cfg.Recipients.Registry["owner"])
}

// jsonSchema declares only the JSON Schema subset this test cares about.
type jsonSchema struct {
	Ref                  string                `json:"$ref"`
	Type                 string                `json:"type"`
	Properties           map[string]jsonSchema `json:"properties"`
	Items                *jsonSchema           `json:"items"`
	Required             []string              `json:"required"`
	Enum                 []string              `json:"enum"`
	Const                string                `json:"const"`
	Pattern              string                `json:"pattern"`
	MinItems             int                   `json:"minItems"`
	AdditionalProperties *json.RawMessage      `json:"additionalProperties"`
	Defs                 map[string]jsonSchema `json:"$defs"`
}

// schemaRequiredFields explicitly anchors the required fields of each definition.
// TOML decoding is lenient about omitted fields (zero values); the real required
// constraints are enforced by LoadConfig (FilePair's plaintext/encrypted,
// GroupConfig's patterns), which Go struct tags cannot express, so they are
// hardcoded here. Any schema requiredness change must update this table.
var schemaRequiredFields = map[string][]string{
	"FilePair":    {"plaintext", "encrypted"},
	"GroupConfig": {"patterns"},
}

// TestSchemaMatchesConfigStructs is the anti-drift tripwire: it aligns the
// Config struct tree with the exported JSON Schema ($defs) definition by
// definition — field set (bidirectional), field types, requiredness, and
// object closedness. When a Go struct changes, update schema/config.cue and
// run mise run schema:export.
func TestSchemaMatchesConfigStructs(t *testing.T) {
	schema := loadJSONSchema(t)
	require.Equal(t, "#/$defs/Config", schema.Ref)
	assertDefMatchesStruct(t, schema.Defs, "Config", reflect.TypeOf(Config{}))
}

// TestSchemaGroupPatternsRequireEntry anchors the minItems constraint on
// patterns: LoadConfig rejects empty arrays, so the schema must too, or
// editors will mark a config that always fails at runtime as valid. Blank
// entries in a mixed array are stripped into empty patterns by go-git per the
// .gitignore trailing-whitespace rules and never match (see the pin test in
// the task package), which is harmless, so only "at least one entry" is
// constrained, not per-entry non-blankness.
func TestSchemaGroupPatternsRequireEntry(t *testing.T) {
	schema := loadJSONSchema(t)
	patterns := schema.Defs["GroupConfig"].Properties["patterns"]
	require.Equal(t, 1, patterns.MinItems, "GroupConfig.patterns should declare minItems:1")
}

func loadJSONSchema(t *testing.T) jsonSchema {
	t.Helper()
	data, err := os.ReadFile(jsonSchemaFile)
	require.NoError(t, err)
	var schema jsonSchema
	require.NoError(t, json.Unmarshal(data, &schema))
	return schema
}

// assertDefMatchesStruct verifies that a JSON Schema definition and a Go struct
// are structurally identical. Naming convention: CUE definition #Foo maps to
// Go struct Foo.
func assertDefMatchesStruct(t *testing.T, defs map[string]jsonSchema, defName string, goType reflect.Type) {
	t.Helper()
	def, ok := defs[defName]
	require.True(t, ok, "JSON Schema missing $defs/%s (Go struct %s); re-export after updating schema/config.cue", defName, goType.Name())
	require.Equal(t, "object", def.Type, "$defs/%s should be object", defName)
	require.NotNil(t, def.AdditionalProperties, "$defs/%s should declare additionalProperties:false to stay closed", defName)
	var closed bool
	require.NoError(t, json.Unmarshal(*def.AdditionalProperties, &closed), "$defs/%s additionalProperties should be a boolean", defName)
	require.False(t, closed, "$defs/%s should be closed (additionalProperties:false)", defName)

	fields := tomlFields(goType)
	require.Equal(t, sortedKeys(fields), sortedKeys(def.Properties), "$defs/%s field set diverges from Go struct %s", defName, goType.Name())
	require.ElementsMatch(t, schemaRequiredFields[defName], def.Required, "$defs/%s required fields diverge from the runtime contract", defName)

	for name, fieldType := range fields {
		assertFieldType(t, defs, defName+"."+name, fieldType, def.Properties[name])
	}
}

// assertFieldType verifies that a field's Go type is compatible with its JSON Schema.
func assertFieldType(t *testing.T, defs map[string]jsonSchema, path string, goType reflect.Type, schema jsonSchema) {
	if goType.Kind() == reflect.Pointer {
		goType = goType.Elem()
	}
	t.Helper()
	switch {
	case goType.Kind() == reflect.String:
		require.True(t, isStringSchema(defs, schema), "%s: Go string should map to string/enum/const (got: %+v)", path, schema)
	case goType.Kind() == reflect.Bool:
		require.Equal(t, "boolean", schema.Type, "%s: Go bool should map to boolean", path)
	case goType.Kind() == reflect.Slice && goType.Elem().Kind() == reflect.String:
		require.Equal(t, "array", schema.Type, "%s: Go []string should map to array", path)
		require.NotNil(t, schema.Items, "%s: array missing items", path)
		require.True(t, isStringSchema(defs, *schema.Items), "%s: []string items should be string (got: %+v)", path, *schema.Items)
	case goType.Kind() == reflect.Slice && goType.Elem().Kind() == reflect.Struct:
		require.Equal(t, "array", schema.Type, "%s: Go []%s should map to array", path, goType.Elem().Name())
		require.NotNil(t, schema.Items, "%s: array missing items", path)
		require.Equal(t, "#/$defs/"+goType.Elem().Name(), schema.Items.Ref, "%s: items should reference the Go struct's definition", path)
		assertDefMatchesStruct(t, defs, goType.Elem().Name(), goType.Elem())
	case goType.Kind() == reflect.Map && goType.Key().Kind() == reflect.String && goType.Elem().Kind() == reflect.String:
		require.Equal(t, "object", schema.Type, "%s: map[string]string should map to object", path)
	case goType.Kind() == reflect.Struct:
		require.Equal(t, "#/$defs/"+goType.Name(), schema.Ref, "%s: Go struct should map to its same-named $defs reference", path)
		assertDefMatchesStruct(t, defs, goType.Name(), goType)
	default:
		t.Fatalf("%s: unhandled Go type %s", path, goType)
	}
}

// isStringSchema reports whether a schema accepts a string value:
// a direct type:string, enum, or const, or indirectly via $ref (e.g. #Format).
func isStringSchema(defs map[string]jsonSchema, schema jsonSchema) bool {
	if schema.Ref != "" {
		resolved, ok := defs[strings.TrimPrefix(schema.Ref, "#/$defs/")]
		if !ok {
			return false
		}
		schema = resolved
	}
	return schema.Type == "string" || len(schema.Enum) > 0 || schema.Const != ""
}

// TestSchemaFormatsMatchRuntime verifies the schema and runtime format contract
// agree, with seal.FormatSpellings as the runtime single source of truth:
//   - FilePair.format's enum covers exactly the spellings the runtime accepts
//     (bidirectional: a missing value lets a workable config be judged invalid,
//     an extra value misleads users)
//   - GroupConfig.format_rules's regex accepts the same spellings and rejects
//     unknown formats
func TestSchemaFormatsMatchRuntime(t *testing.T) {
	schema := loadJSONSchema(t)
	spellings := seal.FormatSpellings()

	formatRef := schema.Defs["FilePair"].Properties["format"].Ref
	formatDef, ok := schema.Defs[strings.TrimPrefix(formatRef, "#/$defs/")]
	require.True(t, ok, "FilePair.format should reference a format definition (got: %q)", formatRef)
	require.ElementsMatch(t, spellings, formatDef.Enum, "#Format enum diverges from the runtime-accepted format spellings")

	rule := schema.Defs["GroupConfig"].Properties["format_rules"]
	require.NotNil(t, rule.Items, "GroupConfig.format_rules missing items")
	pattern, err := regexp.Compile(rule.Items.Pattern)
	require.NoError(t, err)
	for _, format := range spellings {
		require.True(t, pattern.MatchString("*.x="+format), "format_rules regex should accept format %q", format)
	}
	require.False(t, pattern.MatchString("*.x=bogus"), "format_rules regex should not accept unknown formats")
}

// tomlFields collects a struct's toml field names and types, skipping
// toml:"-" runtime fields.
func tomlFields(typ reflect.Type) map[string]reflect.Type {
	fields := map[string]reflect.Type{}
	for i := range typ.NumField() {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(field.Tag.Get("toml"), ",")
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
		}
		fields[name] = field.Type
	}
	return fields
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
