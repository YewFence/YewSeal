package app

import (
	"errors"

	"github.com/YewFence/YewSeal/internal/config"
	"github.com/YewFence/YewSeal/internal/errx"
	"github.com/YewFence/YewSeal/internal/presentation"
	"github.com/YewFence/YewSeal/internal/task"
	"github.com/YewFence/YewSeal/internal/verify"
)

// VerifyRequest is the public interface into the verify feature.
type VerifyRequest struct {
	Targets        []string
	KeyFile        string
	Decrypt        bool // --decrypt: require identity; mismatch = error
	NoDecrypt      bool // --no-decrypt: skip decrypt layer entirely
	SyncSOPSConfig bool
	JSON           bool
	Verbose        bool
}

// VerifyResult carries the verify report and the suggested exit code.
type VerifyResult struct {
	Report   *verify.Report
	ExitCode int
}

// Verify runs the four-layer health check and reports findings.
func Verify(cfg *config.Config, req VerifyRequest) (VerifyResult, error) {
	if req.Decrypt && req.NoDecrypt {
		return VerifyResult{ExitCode: 2}, errx.Usage(errors.New("--decrypt and --no-decrypt are mutually exclusive"))
	}

	selection, err := config.ResolveSelection(cfg, config.SelectionOptions{
		Command: task.ModeVerify,
		Targets: req.Targets,
	})
	if err != nil {
		return VerifyResult{ExitCode: 2}, err
	}

	mode := verify.DecryptAuto
	if req.Decrypt {
		mode = verify.DecryptRequired
	} else if req.NoDecrypt {
		mode = verify.DecryptDisabled
	}

	report, err := verify.Check(selection, config.CurrentDir(cfg), verify.Options{
		KeyFile:         req.KeyFile,
		DecryptMode:     mode,
		CheckSOPSConfig: req.SyncSOPSConfig,
	})
	if err != nil {
		return VerifyResult{ExitCode: 2}, errx.Usage(err)
	}

	exitCode := 0
	if !report.OK() {
		exitCode = 1
	}
	return VerifyResult{Report: report, ExitCode: exitCode}, nil
}

// PrintVerify renders the verify report through the presentation layer.
func PrintVerify(out *presentation.Output, cfg *config.Config, req VerifyRequest) error {
	result, err := Verify(cfg, req)
	if err != nil {
		return err
	}
	return out.Finish(out.VerifyReport(result.Report, presentation.VerifyPrintOptions{
		JSON:    req.JSON,
		Verbose: req.Verbose,
		CWD:     config.CurrentDir(cfg),
	}))
}
