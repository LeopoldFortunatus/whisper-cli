package execx

import (
	"bytes"
	"context"
	"os/exec"
)

type Runner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, name string, args ...string) (stdout []byte, stderr []byte, err error)
}

type OS struct{}

func (OS) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (OS) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
