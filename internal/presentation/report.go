package presentation

import (
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/YewFence/YewSeal/internal/diff"
	"github.com/YewFence/YewSeal/internal/task"
)

func encodeReportJSON(w io.Writer, payload any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

// BatchReportJSON 把一次 encrypt/decrypt 批处理结果渲染为 stdout 的 JSON 报告。
func (o *Output) BatchReportJSON(command string, summary *task.Summary) error {
	if summary == nil {
		return nil
	}
	files := make([]batchFileJSON, 0, len(summary.Results))
	for _, result := range summary.Results {
		plaintext, encrypted := result.SourceFile, result.TargetFile
		if command == "decrypt" {
			// decrypt 的 SourceFile 是密文侧、TargetFile 是明文侧
			plaintext, encrypted = encrypted, plaintext
		}
		files = append(files, batchFileJSON{
			Plaintext: o.path(plaintext),
			Encrypted: o.path(encrypted),
			Status:    string(result.Status),
			Outcome:   string(result.Outcome),
			Warning:   result.Warning,
			Error:     errorString(result.Error),
		})
	}
	return encodeReportJSON(o, batchReportJSON{
		Command: command,
		Summary: batchSummaryJSON{
			Total:            summary.TotalFiles,
			Succeeded:        summary.SuccessCount,
			Skipped:          summary.SkippedCount,
			Failed:           summary.FailedCount,
			Encrypted:        summary.EncryptedCount,
			Unchanged:        summary.UnchangedCount,
			MissingPlaintext: summary.MissingPlaintextCount,
		},
		Files: files,
	})
}

type batchReportJSON struct {
	Command string           `json:"command"`
	Summary batchSummaryJSON `json:"summary"`
	Files   []batchFileJSON  `json:"files"`
}

type batchSummaryJSON struct {
	Total            int `json:"total"`
	Succeeded        int `json:"succeeded"`
	Skipped          int `json:"skipped"`
	Failed           int `json:"failed"`
	Encrypted        int `json:"encrypted"`
	Unchanged        int `json:"unchanged"`
	MissingPlaintext int `json:"missing_plaintext"`
}

type batchFileJSON struct {
	Plaintext string `json:"plaintext"`
	Encrypted string `json:"encrypted"`
	Status    string `json:"status"`
	Outcome   string `json:"outcome,omitempty"`
	Warning   string `json:"warning,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ViewReportJSON 把 view 的解密结果渲染为 stdout 的 JSON 信封；
// binary 格式的正文以 base64 编码，其余按 UTF-8 文本承载。
func (o *Output) ViewReportJSON(path, format string, content []byte) error {
	encoding := "utf-8"
	body := string(content)
	if format == "binary" {
		encoding = "base64"
		body = base64.StdEncoding.EncodeToString(content)
	}
	return encodeReportJSON(o, viewReportJSON{
		Command:  "view",
		Path:     o.path(path),
		Format:   format,
		Encoding: encoding,
		Content:  body,
	})
}

type viewReportJSON struct {
	Command  string `json:"command"`
	Path     string `json:"path"`
	Format   string `json:"format"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
}

// DiffReportJSON 把 diff 的比较结果渲染为 stdout 的 JSON 报告，逐文件内嵌 diff 正文。
func (o *Output) DiffReportJSON(summary diff.Summary) error {
	files := make([]diffFileJSON, 0, len(summary.Results))
	for _, result := range summary.Results {
		entry := diffFileJSON{
			Plaintext: o.path(result.PlaintextFile),
			Encrypted: o.path(result.EncryptedFile),
			Skipped:   string(result.Skipped),
			Error:     errorString(result.Error),
		}
		if result.Error == nil && result.Skipped == "" {
			changed := result.Different
			entry.Changed = &changed
			entry.Diff = result.Diff
		}
		files = append(files, entry)
	}
	return encodeReportJSON(o, diffReportJSON{
		Command: "diff",
		Summary: diffSummaryJSON{
			Total:        len(summary.Results),
			Compared:     summary.ComparedCount,
			MissingInput: summary.MissingInputCount,
			NoIdentity:   summary.NoIdentityCount,
			Failed:       summary.FailedCount,
		},
		Files: files,
	})
}

type diffReportJSON struct {
	Command string          `json:"command"`
	Summary diffSummaryJSON `json:"summary"`
	Files   []diffFileJSON  `json:"files"`
}

type diffSummaryJSON struct {
	Total        int `json:"total"`
	Compared     int `json:"compared"`
	MissingInput int `json:"missing_input"`
	NoIdentity   int `json:"no_identity"`
	Failed       int `json:"failed"`
}

type diffFileJSON struct {
	Plaintext string `json:"plaintext"`
	Encrypted string `json:"encrypted"`
	Changed   *bool  `json:"changed,omitempty"`
	Diff      string `json:"diff,omitempty"`
	Skipped   string `json:"skipped,omitempty"`
	Error     string `json:"error,omitempty"`
}

// InitReport 汇总一次 init 的产物，由 project 层填充。
type InitReport struct {
	Kept         bool
	ConfigFile   string
	KeyFile      string
	SOPSConfig   bool
	Mappings     []InitMapping
	ExampleFiles []string
}

type InitMapping struct {
	Plaintext  string
	Encrypted  string
	Format     string
	Recipients []string
}

// InitReportJSON 把 init 的产物渲染为 stdout 的 JSON 报告。
func (o *Output) InitReportJSON(report InitReport) error {
	payload := initReportJSON{
		Command:      "init",
		Kept:         report.Kept,
		ConfigFile:   report.ConfigFile,
		KeyFile:      report.KeyFile,
		SOPSConfig:   report.SOPSConfig,
		ExampleFiles: report.ExampleFiles,
	}
	if !report.Kept {
		payload.Mappings = make([]initMappingJSON, 0, len(report.Mappings))
		for _, mapping := range report.Mappings {
			payload.Mappings = append(payload.Mappings, initMappingJSON{
				Plaintext:  o.path(mapping.Plaintext),
				Encrypted:  o.path(mapping.Encrypted),
				Format:     mapping.Format,
				Recipients: mapping.Recipients,
			})
		}
		if payload.ExampleFiles == nil {
			payload.ExampleFiles = []string{}
		}
	}
	return encodeReportJSON(o, payload)
}

type initReportJSON struct {
	Command      string            `json:"command"`
	Kept         bool              `json:"kept,omitempty"`
	ConfigFile   string            `json:"config_file,omitempty"`
	KeyFile      string            `json:"key_file,omitempty"`
	SOPSConfig   bool              `json:"sops_config,omitempty"`
	Mappings     []initMappingJSON `json:"mappings,omitempty"`
	ExampleFiles []string          `json:"example_files,omitempty"`
}

type initMappingJSON struct {
	Plaintext  string   `json:"plaintext"`
	Encrypted  string   `json:"encrypted"`
	Format     string   `json:"format"`
	Recipients []string `json:"recipients"`
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
