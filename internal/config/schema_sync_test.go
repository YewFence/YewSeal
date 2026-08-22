package config

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

// schema/config.cue 与 Go struct 的对齐锚点,详见 schema/config.cue 头部注释。
const (
	schemaFile  = "../../schema/config.cue"
	exampleFile = "../../schema/example.yewseal.toml"
)

// TestExampleConfigStrictUnmarshal 以严格模式反序列化全字段示例,
// 保证 schema 与 example 中的每个字段都在 Go Config 中真实存在。
func TestExampleConfigStrictUnmarshal(t *testing.T) {
	data, err := os.ReadFile(exampleFile)
	require.NoError(t, err)

	var cfg Config
	decoder := toml.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&cfg))

	// 抽查关键字段,确认 example 内容映射到了正确位置,而不仅仅是"不报错"
	require.Len(t, cfg.Encryption.Files, 2)
	require.Equal(t, "wrangler.toml", cfg.Encryption.Files[0].PlaintextPath)
	require.Equal(t, ".dev.vars", cfg.Encryption.Files[1].PlaintextPath)
	require.Equal(t, "env", cfg.Encryption.Files[1].Format)

	require.Len(t, cfg.Encryption.Groups, 1)
	require.Equal(t, []string{"config/**/*.toml", "secrets/*.yaml"}, cfg.Encryption.Groups[0].Patterns)
	require.Equal(t, []string{"*.dev.vars=env", "**/*.conf=ini"}, cfg.Encryption.Groups[0].FormatRules)
	require.True(t, cfg.Encryption.Groups[0].UnknownAsBinary)

	require.Equal(t, ".age/keys.txt", cfg.Key.FilePath)
	require.NotEmpty(t, cfg.Key.PublicKey)

	require.Equal(t, "infisical", cfg.Sync.Provider)
	require.Equal(t, "AGE_KEY_FILE", cfg.Sync.SecretName)
	require.Equal(t, "/yewseal", cfg.Sync.Path)
	require.Equal(t, "production", cfg.Sync.Environment)
}

// TestSchemaCoversAllTomlFields 是防漂移 tripwire:
// Go struct 新增或重命名 toml 字段时,必须同步更新 schema/config.cue。
func TestSchemaCoversAllTomlFields(t *testing.T) {
	schema, err := os.ReadFile(schemaFile)
	require.NoError(t, err)

	for name := range collectTomlFieldNames(reflect.TypeOf(Config{})) {
		pattern := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(name) + `[?!]?:`)
		require.True(t, pattern.Match(schema), "schema/config.cue 缺少 toml 字段 %q", name)
	}
}

// collectTomlFieldNames 递归收集 Config 及其嵌套 struct 的全部 toml 字段名,
// 跳过 toml:"-" 的运行时字段。
func collectTomlFieldNames(typ reflect.Type) map[string]bool {
	names := map[string]bool{}
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		t = indirectType(t)
		if t.Kind() != reflect.Struct {
			return
		}
		for i := range t.NumField() {
			field := t.Field(i)
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
			names[name] = true
			walk(field.Type)
		}
	}
	walk(typ)
	return names
}

func indirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		t = t.Elem()
	}
	return t
}
