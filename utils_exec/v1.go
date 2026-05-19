package utils_exec

import (
	"bytes"
	"context"
	"os/exec"
)

type ExecInfo struct {
	Command *ExecCommand
	Result  *ExecResult
}
type ExecCommand struct {
	Name string
	Args []string
}
type ExecResult struct {
	Stdout bytes.Buffer
	Stderr bytes.Buffer
	Err    error
}

func (ei *ExecInfo) GetResult() *ExecResult {
	return ei.Result
}

func (er *ExecResult) IsError() bool {
	if er == nil || er.Err != nil {
		return true
	}
	return false
}

func (er *ExecResult) IsStderr() bool {
	if er == nil || er.Stderr.Len() > 0 {
		return true
	}
	return false
}

func (er *ExecResult) Success() bool {
	if !er.IsError() && !er.IsStderr() {
		return true
	}
	return false
}

func (er *ExecResult) GetStdoutString() string {
	return er.Stdout.String()
}

func (er *ExecResult) GetStderrString() string {
	return er.Stderr.String()
}

// ExecCmd 执行外部命令；ctx 取消时会终止子进程。
func ExecCmd(ctx context.Context, name string, args []string) *ExecInfo {
	res := &ExecInfo{
		Command: &ExecCommand{
			Name: name,
			Args: args,
		},
		Result: &ExecResult{},
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &res.Result.Stdout
	cmd.Stderr = &res.Result.Stderr
	res.Result.Err = cmd.Run()

	return res
}
