package autopilot

import (
	"context"
	"io"
)

type SessionRequest struct {
	Prompt      string
	Model       string
	WorkingDir  string
	AutoApprove bool
}

type SessionRunner interface {
	Preflight(ctx context.Context, req SessionRequest) error
	Run(ctx context.Context, req SessionRequest, stdout, stderr io.Writer) error
}

func newSessionRunner(opts PipelineOptions) SessionRunner {
	if opts.Backend == BackendOpencode {
		return newOpencodeRunner(opts)
	}
	return newCLIRunner(opts)
}
