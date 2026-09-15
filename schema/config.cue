// Package schema is the authoritative schema for .yewseal.toml, kept in sync
// with the Go structs in internal/config.
//
// Three-way anchoring prevents drift:
//   - mise run schema:check uses cue vet to validate that example.yewseal.toml
//     conforms to this schema
//   - The Go tests in internal/config strict-unmarshal the same example
//     (fields unknown to Go fail the test)
//   - A reflection tripwire test aligns the Go structs with the exported JSON
//     Schema field by field (parents, types, requiredness); the diff inside
//     schema:check guarantees the export matches this file
package schema

// #Config is the top-level structure of .yewseal.toml. Every section is
// optional. The encryption and plan-selection stages require paths to come
// from files/groups and require the final authorized set to be non-empty.
#Config: {
	encryption?: #EncryptionConfig
	recipients?: #RecipientConfig
}

// #Format lists the encrypted file formats supported by YewSeal.
// Beyond the canonical names, runtime aliases are accepted
// (yml→yaml, dotenv→env, bin→binary), kept consistent with
// internal/seal.FormatSpellings (enforced by a Go test); the runtime
// normalizes aliases to canonical names.
#Format: "toml" | "yaml" | "yml" | "json" | "env" | "dotenv" | "ini" | "binary" | "bin"

// #EncryptionConfig defines the encrypted file mapping.
#EncryptionConfig: {
	// Explicit plaintext/encrypted file pairs. Every path processed at
	// runtime must come from here or from groups.
	files?: [...#FilePair]

	// Groups of encrypted files matched in bulk by glob patterns.
	groups?: [...#GroupConfig]
}

// #RecipientConfig defines the public recipient authorization policy.
// The registry contains public Age recipients only, never private keys.
#RecipientConfig: {
	// Default alias set. When absent, every file/group must declare its
	// own recipients explicitly.
	defaults?: [string, ...string]

	// Mapping from alias to an Age recipient public key.
	registry?: {[string]: string}
}

// #FilePair defines a plaintext/encrypted file pair.
#FilePair: {
	// Plaintext file path, used as the input of encrypt and the output of
	// decrypt. Relative paths resolve against the directory containing
	// this config file; absolute paths take effect literally but bind the
	// config to a single machine (it breaks after a clone, in CI, or after
	// moving the project directory), so they are not recommended. ~ is not
	// expanded; Windows absolute paths must include a drive letter.
	plaintext!: string

	// Encrypted file path, used as the output of encrypt and the input of
	// decrypt. Resolution rules are the same as for plaintext.
	encrypted!: string

	// Overrides format detection, for files with non-standard extensions
	// (e.g. .dev.vars).
	format?: #Format

	// Authorized alias set. When omitted, inherits from the group or the
	// top-level defaults.
	recipients?: [string, ...string]
}

// #GroupConfig defines a set of encrypted files matched by patterns.
#GroupConfig: {
	// List of glob patterns, e.g. "config/**/*.toml". Required: a group
	// without patterns would implicitly sweep every config-like file in
	// the config directory, so it is rejected.
	// At least one entry (aligned with LoadConfig's "at least one
	// non-blank pattern" constraint; blank entries in a mixed array are
	// stripped into empty patterns by go-git per the .gitignore
	// trailing-whitespace rules and never match, which is harmless, so
	// entries are not constrained individually).
	// Encryption always excludes *.enc.* files in YewSeal protocol format
	// and the encrypted paths of explicit FilePairs:
	patterns!: [string, ...string]

	// Format override rules with the syntax "<glob>=<format>", e.g.
	// "*.dev.vars=env".
	// format only accepts lowercase (the Go runtime is case-insensitive;
	// the schema nudges toward the canonical style). Like #Format, it
	// accepts the yml/dotenv/bin aliases.
	format_rules?: [...=~"^.+=(toml|yaml|yml|json|env|dotenv|ini|binary|bin)$"]

	// Files whose format cannot be detected are treated as binary.
	unknown_as_binary?: bool

	// Authorized alias set for scan results. When omitted, inherits from
	// the top-level defaults.
	recipients?: [string, ...string]
}

// Top-level default reference, so cue vet/export can use it directly:
//   cue vet ./schema example.yewseal.toml -d '#Config'
#Config
