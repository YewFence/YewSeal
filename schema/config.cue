// Package schema 是 .yewseal.toml 的权威 schema,与 internal/config 的 Go struct 保持同步。
//
// 三向锚定防漂移:
//   - mise run schema:check 用 cue vet 校验 example.yewseal.toml 符合本 schema
//   - internal/config 的 Go 测试 strict-unmarshal 同一 example(Go 不认识的字段会报错)
//   - 反射 tripwire 测试要求 Go struct 的每个 toml 字段名都出现在本文件中
package schema

// #Config 是 .yewseal.toml 的顶层结构。所有段落都可缺省,
// 缺省时由 CLI 参数、环境变量或内置默认值兜底。
#Config: {
	encryption?: #EncryptionConfig
	key?:        #KeyConfig
	sync?:       #SyncConfig
}

// #Format 是 YewSeal 支持的加密文件格式。
#Format: "toml" | "yaml" | "json" | "env" | "ini" | "binary"

// #EncryptionConfig 定义加密文件映射。
#EncryptionConfig: {
	// 显式的明文/加密文件对。缺省且无 groups 时使用
	// 默认映射 wrangler.toml <-> wrangler.enc.toml。
	files?: [...#FilePair]

	// 按 glob 模式批量匹配的加密文件组。
	groups?: [...#GroupConfig]
}

// #FilePair 定义一对明文/加密文件映射。
#FilePair: {
	// 明文文件路径,用作 encrypt 的输入和 decrypt 的输出。
	plaintext!: string

	// 加密文件路径,用作 encrypt 的输出和 decrypt 的输入。
	encrypted!: string

	// 覆盖文件格式探测,用于扩展名不标准的文件(如 .dev.vars)。
	format?: #Format
}

// #GroupConfig 定义一组按模式匹配的加密文件。
#GroupConfig: {
	// glob 模式列表,如 "config/**/*.toml"。
	patterns!: [...string]

	// 格式覆盖规则,语法为 "<glob>=<format>",如 "*.dev.vars=env"。
	// format 仅接受小写(Go 运行时对大小写宽容,schema 引导规范写法)。
	format_rules?: [...=~"^.+=(toml|yaml|json|env|ini|binary)$"]

	// 无法探测格式的文件按 binary 处理。
	unknown_as_binary?: bool
}

// #KeyConfig 定义 Age 密钥位置。切勿把私钥内容写进配置文件。
#KeyConfig: {
	// Age 私钥文件路径。
	file_path?: string | *".age/keys.txt"

	// Age 公钥,用于加密(可以安全提交)。
	public_key?: string
}

// #SyncConfig 定义 Age 密钥同步设置。
#SyncConfig: {
	// 密钥管理 Provider 名称。当前仅支持 infisical,
	// 新增 Provider 时需同步扩展此枚举。
	provider?: "infisical"

	// Provider 侧的项目标识符,sync 命令使用。
	project_id?: string

	// Age 私钥文件在 Provider 侧的 secret 名称。
	secret_name?: string | *"AGE_KEY_FILE"

	// Provider 侧的远程路径/目录。
	path?: string

	// Provider 侧的环境名称。
	environment?: string
}

// 顶层默认引用,便于 cue vet/export 直接使用:
//   cue vet ./schema example.yewseal.toml -d '#Config'
#Config
