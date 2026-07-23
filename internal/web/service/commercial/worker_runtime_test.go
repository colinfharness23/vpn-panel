package commercial

import (
	"errors"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

type fakeClientRuntimeConverger struct {
	restartRequired bool
	restartCalls    int
	force           bool
	err             error
}

func (f *fakeClientRuntimeConverger) SetToNeedRestart() {
	f.restartRequired = true
}

func (f *fakeClientRuntimeConverger) RestartXray(force bool) error {
	f.restartCalls++
	f.force = force
	return f.err
}

func TestApplyClientRuntimeMutationKeepsRequiredRestartSignal(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_ENV", "test")
	xrayService := &service.XrayService{}
	_ = xrayService.IsNeedRestartAndSetFalse()

	if err := NewWorker().applyClientRuntimeMutation(true); err != nil {
		t.Fatalf("apply client runtime mutation: %v", err)
	}
	if !xrayService.IsNeedRestartAndSetFalse() {
		t.Fatal("required client runtime restart signal was discarded")
	}
	if err := NewWorker().applyClientRuntimeMutation(false); err != nil {
		t.Fatalf("no-op client runtime mutation: %v", err)
	}
	if xrayService.IsNeedRestartAndSetFalse() {
		t.Fatal("no-op client mutation scheduled an unnecessary restart")
	}
}

func TestApplyClientRuntimeMutationConvergesProductionBeforeProvisionCompletes(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_ENV", "production")
	runtime := &fakeClientRuntimeConverger{}
	worker := NewWorker()
	worker.xray = runtime

	if err := worker.applyClientRuntimeMutation(true); err != nil {
		t.Fatalf("converge production client runtime: %v", err)
	}
	if !runtime.restartRequired || runtime.restartCalls != 1 || runtime.force {
		t.Fatalf("unexpected convergence: %+v", runtime)
	}
}

func TestApplyClientRuntimeMutationReturnsProductionRestartFailure(t *testing.T) {
	t.Setenv("XUI_COMMERCIAL_ENV", "production")
	want := errors.New("xray restart failed")
	runtime := &fakeClientRuntimeConverger{err: want}
	worker := NewWorker()
	worker.xray = runtime

	if err := worker.applyClientRuntimeMutation(true); !errors.Is(err, want) {
		t.Fatalf("restart failure = %v, want %v", err, want)
	}
}
