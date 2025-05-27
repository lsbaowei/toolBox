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

func (er *ExecResult) IsError() bool {
	if er.Err == nil {
		return false
	}
	return true
}

func (er *ExecResult) IsStderr() bool {
	if er.Stderr.Len() == 0 {
		return false
	}
	return true
}

func (er *ExecResult) Success() bool {
	if er.IsError() || er.IsStderr() {
		return false
	}
	return true
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

	// 要执行的命令
	cmd := exec.Command(name, args...) // 你可以替换成任何命令，如 "ffmpeg", "curl", etc.

	// 捕获输出（标准输出 & 标准错误）  var stdout, stderr bytes.Buffer
	cmd.Stdout = &res.Result.Stdout
	cmd.Stderr = &res.Result.Stderr

	// 执行命令
	res.Result.Err = cmd.Run()

	//// 输出结果
	//fmt.Println("STDOUT:\n", stdout.String())
	//fmt.Println("STDERR:\n", stderr.String())
	//
	//// 检查是否执行成功
	//if err != nil {
	//	fmt.Printf("Command failed with error: %v\n", err)
	//} else {
	//	fmt.Println("Command executed successfully.")
	//}

	return res
}
