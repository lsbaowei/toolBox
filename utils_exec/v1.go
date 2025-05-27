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

func ExecCmd(ctx context.Context, name string, args []string) *ExecInfo {
	res := &ExecInfo{
		Command: &ExecCommand{
			Name: name,
			Args: args,
		},
		Result: &ExecResult{
			Stdout: bytes.Buffer{},
			Stderr: bytes.Buffer{},
			Err:    nil,
		},
	}

	errCh := make(chan error, 1)

	go func() {
		cmd := exec.Command(name, args...) // 要执行的命令 // 你可以替换成任何命令，如 "ffmpeg", "curl", etc.

		// 捕获输出（标准输出 & 标准错误）  var stdout, stderr bytes.Buffer
		cmd.Stdout = &res.Result.Stdout
		cmd.Stderr = &res.Result.Stderr

		// 执行命令
		errCh <- cmd.Run() // 命令执行完成后发送信号
	}()

	select {
	case <-ctx.Done():
		res.Result.Err = ctx.Err()
	case err := <-errCh:
		res.Result.Err = err // 这里可以添加其他逻辑，如果需要在执行命令前做一些事情
	}

	return res
}
