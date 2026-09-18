package agekey

import (
	"errors"
	"io/fs"
	"os"

	"filippo.io/age"

	"github.com/YewFence/YewSeal/internal/errx"
)

// ResolvedIdentity 是一个已解析的 Age 私钥身份及其推导出的公钥。
type ResolvedIdentity struct {
	Secret    string
	PublicKey string
}

// IdentitySources 报告身份解析链的 first-win 结果：生效来源、实际存在
// 却被短路的来源、以及生效来源内的全部身份（按首次出现序去重）。
type IdentitySources struct {
	Source     string
	Shadowed   []string
	Identities []ResolvedIdentity
	Warnings   []string
}

type identityLayer struct {
	label   string
	present func() bool
	load    func() (IdentityBundle, error)
}

var identitySourceOptions = []string{"--key-file", "YEWSEAL_AGE_IDENTITIES", "SOPS_AGE_KEY", "SOPS_AGE_KEY_FILE", "SOPS_AGE_KEY_CMD", "or .age/keys.txt"}

// ResolveIdentitySources 按 GetIdentityBundle 的同一优先级链解析身份，
// 并额外报告生效来源与被短路的候选来源。来源标识为扁平字符串：文件
// 来源是 "file:<path>"，值或命令来源是 "env:<NAME>"。
func ResolveIdentitySources(keyFile string) (IdentitySources, error) {
	layers := identityLayers(keyFile)
	for index, layer := range layers {
		if !layer.present() {
			continue
		}
		bundle, err := layer.load()
		if err != nil {
			return IdentitySources{}, err
		}
		identities, err := withPublicKeys(bundle)
		if err != nil {
			return IdentitySources{}, err
		}
		return IdentitySources{
			Source:     layer.label,
			Shadowed:   collectShadowed(layers[index+1:]),
			Identities: identities,
			Warnings:   bundle.Warnings(),
		}, nil
	}
	return IdentitySources{}, &errx.AgeKeyNotFoundError{Options: identitySourceOptions}
}

func identityLayers(keyFile string) []identityLayer {
	return []identityLayer{
		{
			label:   "file:" + keyFile,
			present: func() bool { return keyFile != "" },
			load:    func() (IdentityBundle, error) { return readIdentityBundle(keyFile) },
		},
		{
			label:   "env:YEWSEAL_AGE_IDENTITIES",
			present: func() bool { return os.Getenv("YEWSEAL_AGE_IDENTITIES") != "" },
			load: func() (IdentityBundle, error) {
				return parseIdentityFile(os.Getenv("YEWSEAL_AGE_IDENTITIES"))
			},
		},
		{
			label:   "env:SOPS_AGE_KEY",
			present: func() bool { return os.Getenv("SOPS_AGE_KEY") != "" },
			load: func() (IdentityBundle, error) {
				return parseIdentityFile(os.Getenv("SOPS_AGE_KEY"))
			},
		},
		{
			label: "file:" + os.Getenv("SOPS_AGE_KEY_FILE"),
			present: func() bool {
				path := os.Getenv("SOPS_AGE_KEY_FILE")
				if path == "" {
					return false
				}
				_, err := os.Stat(path)
				return err == nil || !errors.Is(err, fs.ErrNotExist)
			},
			load: func() (IdentityBundle, error) {
				return readIdentityBundle(os.Getenv("SOPS_AGE_KEY_FILE"))
			},
		},
		{
			label:   "env:SOPS_AGE_KEY_CMD",
			present: func() bool { return os.Getenv("SOPS_AGE_KEY_CMD") != "" },
			load: func() (IdentityBundle, error) {
				value, err := runKeyCommand()
				if err != nil {
					return IdentityBundle{}, err
				}
				return parseIdentityFile(value)
			},
		},
		{
			label: "file:.age/keys.txt",
			present: func() bool {
				_, err := os.Stat(".age/keys.txt")
				return err == nil || !errors.Is(err, fs.ErrNotExist)
			},
			load: func() (IdentityBundle, error) { return readIdentityBundle(".age/keys.txt") },
		},
	}
}

func collectShadowed(layers []identityLayer) []string {
	var shadowed []string
	for _, layer := range layers {
		if layer.present() {
			shadowed = append(shadowed, layer.label)
		}
	}
	return shadowed
}

func withPublicKeys(bundle IdentityBundle) ([]ResolvedIdentity, error) {
	identities := bundle.Identities()
	resolved := make([]ResolvedIdentity, 0, len(identities))
	for _, secret := range identities {
		identity, err := age.ParseX25519Identity(secret)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, ResolvedIdentity{Secret: secret, PublicKey: identity.Recipient().String()})
	}
	return resolved, nil
}
