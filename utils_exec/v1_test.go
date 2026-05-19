package utils_exec

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"
)

func TestExecCmdSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix test")
	}
	ctx := context.Background()
	info := ExecCmd(ctx, "echo", []string{"hello"})
	if info.Result.IsError() {
		t.Fatalf("err: %v stderr: %s", info.Result.Err, info.Result.GetStderrString())
	}
}

func TestExecCmdContextCancel(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	info := ExecCmd(ctx, "sleep", []string{"10"})
	if info.Result.Err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(info.Result.Err, context.DeadlineExceeded) || errors.Is(info.Result.Err, context.Canceled) {
		return
	}
	// CommandContext 超时时子进程可能收到 SIGKILL
	if info.Result.Err.Error() == "signal: killed" {
		return
	}
	t.Fatalf("got %v", info.Result.Err)
}
