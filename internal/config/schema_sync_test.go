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

// schema/ 与 Go struct 的对齐锚点,详见 schema/config.cue 头部注释。
// yewseal.schema.json 由 config.cue 导出,mise run schema:check 保证导出新鲜度,
// 因此校验导出的 JSON Schema 等价于校验 config.cue 的结构。
const (
	exampleFile    = "../../schema/example.yewseal.toml"
	jsonSchemaFile = "../../schema/yewseal.schema.json"
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

// jsonSchema 只声明本测试关心的 JSON Schema 子集。
type jsonSchema struct {
	Ref                  string                `json:"$ref"`
	Type                 string                `json:"type"`
	Properties           map[string]jsonSchema `json:"properties"`
	Items                *jsonSchema           `json:"items"`
	Required             []string              `json:"required"`
	Enum                 []string              `json:"enum"`
	Const                string                `json:"const"`
	Pattern              string                `json:"pattern"`
	AdditionalProperties *bool                 `json:"additionalProperties"`
	Defs                 map[string]jsonSchema `json:"$defs"`
}

// schemaRequiredFields 显式锚定各定义的必填字段。
// TOML 解码对缺省字段宽容(零值),真正的必填约束由 LoadConfig 校验
// (目前只有 FilePair 的 plaintext/encrypted);Go struct 标签无法表达这一点,
// 因此在此硬编码,schema 必填性变化必须同步更新本表。
var schemaRequiredFields = map[string][]string{
	"FilePair": {"plaintext", "encrypted"},
}

// TestSchemaMatchesConfigStructs 是防漂移 tripwire:
// 把 Config struct 树与导出的 JSON Schema($defs)逐定义结构对齐——
// 字段集合(双向)、字段类型、必填性与对象闭合性。
// Go struct 变更时同步更新 schema/config.cue 并执行 mise run schema:export。
func TestSchemaMatchesConfigStructs(t *testing.T) {
	schema := loadJSONSchema(t)
	require.Equal(t, "#/$defs/Config", schema.Ref)
	assertDefMatchesStruct(t, schema.Defs, "Config", reflect.TypeOf(Config{}))
}

func loadJSONSchema(t *testing.T) jsonSchema {
	t.Helper()
	data, err := os.ReadFile(jsonSchemaFile)
	require.NoError(t, err)
	var schema jsonSchema
	require.NoError(t, json.Unmarshal(data, &schema))
	return schema
}

// assertDefMatchesStruct 校验一个 JSON Schema 定义与 Go struct 的结构一致。
// 命名约定:CUE 定义 #Foo 对应 Go struct Foo。
func assertDefMatchesStruct(t *testing.T, defs map[string]jsonSchema, defName string, goType reflect.Type) {
	t.Helper()
	def, ok := defs[defName]
	require.True(t, ok, "JSON Schema 缺少 $defs/%s(Go struct %s);更新 schema/config.cue 后需重新导出", defName, goType.Name())
	require.Equal(t, "object", def.Type, "$defs/%s 应为 object", defName)
	require.NotNil(t, def.AdditionalProperties, "$defs/%s 应声明 additionalProperties:false 保持闭合", defName)
	require.False(t, *def.AdditionalProperties, "$defs/%s 应闭合(additionalProperties:false)", defName)

	fields := tomlFields(goType)
	require.Equal(t, sortedKeys(fields), sortedKeys(def.Properties), "$defs/%s 与 Go struct %s 的字段集合不一致", defName, goType.Name())
	require.ElementsMatch(t, schemaRequiredFields[defName], def.Required, "$defs/%s 的必填字段与运行时契约不一致", defName)

	for name, fieldType := range fields {
		assertFieldType(t, defs, defName+"."+name, fieldType, def.Properties[name])
	}
}

// assertFieldType 校验单个字段的 Go 类型与 JSON Schema 声明兼容。
func assertFieldType(t *testing.T, defs map[string]jsonSchema, path string, goType reflect.Type, schema jsonSchema) {
	t.Helper()
	switch {
	case goType.Kind() == reflect.String:
		require.True(t, isStringSchema(defs, schema), "%s: Go string 应对应 string/enum/const(当前: %+v)", path, schema)
	case goType.Kind() == reflect.Bool:
		require.Equal(t, "boolean", schema.Type, "%s: Go bool 应对应 boolean", path)
	case goType.Kind() == reflect.Slice && goType.Elem().Kind() == reflect.String:
		require.Equal(t, "array", schema.Type, "%s: Go []string 应对应 array", path)
		require.NotNil(t, schema.Items, "%s: array 缺少 items", path)
		require.True(t, isStringSchema(defs, *schema.Items), "%s: []string 的 items 应为 string(当前: %+v)", path, *schema.Items)
	case goType.Kind() == reflect.Slice && goType.Elem().Kind() == reflect.Struct:
		require.Equal(t, "array", schema.Type, "%s: Go []%s 应对应 array", path, goType.Elem().Name())
		require.NotNil(t, schema.Items, "%s: array 缺少 items", path)
		require.Equal(t, "#/$defs/"+goType.Elem().Name(), schema.Items.Ref, "%s: items 应引用 Go struct 对应的定义", path)
		assertDefMatchesStruct(t, defs, goType.Elem().Name(), goType.Elem())
	case goType.Kind() == reflect.Struct:
		require.Equal(t, "#/$defs/"+goType.Name(), schema.Ref, "%s: Go struct 应对应同名 $defs 引用", path)
		assertDefMatchesStruct(t, defs, goType.Name(), goType)
	default:
		t.Fatalf("%s: 未处理的 Go 类型 %s", path, goType)
	}
}

// isStringSchema 判断 schema 是否接受字符串值:
// 直接的 type:string、enum、const,或经 $ref 间接满足(如 #Format)。
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

// runtimeFormats 是 internal/seal parseFormat 接受的全部格式拼写(规范名+别名)。
// parseFormat 新增或移除拼写时必须同步更新本表与 schema/config.cue。
var runtimeFormats = []string{"toml", "yaml", "yml", "json", "env", "dotenv", "ini", "binary", "bin"}

// TestSchemaFormatsMatchRuntime 校验 schema 与运行时的格式契约一致:
//   - FilePair.format 引用的枚举恰好覆盖运行时接受的拼写
//     (schema 拒绝运行时接受的值会让可执行配置被判无效;反之则误导用户)
//   - GroupConfig.format_rules 的正则接受同样的拼写,并拒绝未知格式
func TestSchemaFormatsMatchRuntime(t *testing.T) {
	schema := loadJSONSchema(t)

	// 先确认硬编码的 runtimeFormats 与运行时实现一致,防止运行时单方面漂移
	for _, format := range runtimeFormats {
		_, ok := seal.NormalizeFormatOverride(format)
		require.True(t, ok, "运行时不再接受格式 %q,同步更新 runtimeFormats 与 schema/config.cue", format)
	}

	formatRef := schema.Defs["FilePair"].Properties["format"].Ref
	formatDef, ok := schema.Defs[strings.TrimPrefix(formatRef, "#/$defs/")]
	require.True(t, ok, "FilePair.format 应引用格式定义(当前: %q)", formatRef)
	require.ElementsMatch(t, runtimeFormats, formatDef.Enum, "#Format 枚举与运行时接受的格式拼写不一致")

	rule := schema.Defs["GroupConfig"].Properties["format_rules"]
	require.NotNil(t, rule.Items, "GroupConfig.format_rules 缺少 items")
	pattern, err := regexp.Compile(rule.Items.Pattern)
	require.NoError(t, err)
	for _, format := range runtimeFormats {
		require.True(t, pattern.MatchString("*.x="+format), "format_rules 正则应接受格式 %q", format)
	}
	require.False(t, pattern.MatchString("*.x=bogus"), "format_rules 正则不应接受未知格式")
}

// tomlFields 收集 struct 的 toml 字段名与类型,跳过 toml:"-" 的运行时字段。
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
