package chat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// fakeTool is a configurable ManagedTool used to exercise the Registry's
// cross-cutting behaviour without depending on real adapters.
type fakeTool struct {
	name    string
	cap     Capability
	phases  []Phase
	timeout time.Duration
	runFn   func(ctx context.Context, args string) (string, error)
}

func (f *fakeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: f.name, Desc: "fake"}, nil
}
func (f *fakeTool) InvokableRun(ctx context.Context, args string, _ ...tool.Option) (string, error) {
	return f.runFn(ctx, args)
}
func (f *fakeTool) Capability() Capability { return f.cap }
func (f *fakeTool) AllowedPhases() []Phase { return f.phases }
func (f *fakeTool) Timeout() time.Duration { return f.timeout }

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func runOnce(t *testing.T, ft *fakeTool, args string) string {
	t.Helper()
	r := NewRegistry(silentLogger())
	r.Register(ft)
	tools, err := r.BuildEinoTools()
	if err != nil {
		t.Fatalf("BuildEinoTools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("want 1 tool, got %d", len(tools))
	}
	inv, ok := tools[0].(tool.InvokableTool)
	if !ok {
		t.Fatalf("wrapped tool is not InvokableTool")
	}
	out, err := inv.InvokableRun(context.Background(), args)
	if err != nil {
		t.Fatalf("wrapped run returned err (should always be nil): %v", err)
	}
	return out
}

func TestRegistry_SuccessPassthrough(t *testing.T) {
	ft := &fakeTool{
		name:  "echo",
		cap:   CapRead,
		runFn: func(_ context.Context, args string) (string, error) { return `{"ok":true}`, nil },
	}
	out := runOnce(t, ft, "")
	if out != `{"ok":true}` {
		t.Fatalf("success payload mutated: %s", out)
	}
}

func TestRegistry_ToolErrorEnvelope(t *testing.T) {
	ft := &fakeTool{
		name: "boom",
		cap:  CapMutate,
		runFn: func(_ context.Context, _ string) (string, error) {
			return "", NewToolError(CodeNotFound, "no such record").
				WithHint("call list_entries first").
				WithContext(map[string]any{"id": "x"})
		},
	}
	out := runOnce(t, ft, "")
	var env struct {
		Error ToolError `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("envelope not valid JSON: %v (%s)", err, out)
	}
	if env.Error.Code != CodeNotFound {
		t.Fatalf("code=%s, want %s", env.Error.Code, CodeNotFound)
	}
	if env.Error.Hint != "call list_entries first" {
		t.Fatalf("hint lost: %q", env.Error.Hint)
	}
	if !env.Error.Recoverable {
		t.Fatalf("CodeNotFound should be recoverable")
	}
}

func TestRegistry_OpaqueErrorBucketed(t *testing.T) {
	ft := &fakeTool{
		name: "raw",
		cap:  CapRead,
		runFn: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("kaboom")
		},
	}
	out := runOnce(t, ft, "")
	if !strings.Contains(out, `"code":"internal_error"`) {
		t.Fatalf("opaque errors should map to internal_error: %s", out)
	}
}

func TestRegistry_PanicRecovered(t *testing.T) {
	ft := &fakeTool{
		name: "panicker",
		cap:  CapMutate,
		runFn: func(_ context.Context, _ string) (string, error) {
			panic("nil deref")
		},
	}
	out := runOnce(t, ft, "")
	if !strings.Contains(out, `"code":"internal_error"`) {
		t.Fatalf("panic should yield internal_error envelope: %s", out)
	}
	if !strings.Contains(out, "panicked") {
		t.Fatalf("panic message not preserved: %s", out)
	}
}

func TestRegistry_TimeoutClassified(t *testing.T) {
	ft := &fakeTool{
		name:    "slow",
		cap:     CapRead,
		timeout: 20 * time.Millisecond,
		runFn: func(ctx context.Context, _ string) (string, error) {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(200 * time.Millisecond):
				return "{}", nil
			}
		},
	}
	out := runOnce(t, ft, "")
	if !strings.Contains(out, `"code":"timeout"`) {
		t.Fatalf("expected timeout code, got %s", out)
	}
}

func TestRegistry_DuplicateNameRejected(t *testing.T) {
	r := NewRegistry(silentLogger())
	r.Register(&fakeTool{name: "dup", runFn: func(context.Context, string) (string, error) { return "", nil }})
	r.Register(&fakeTool{name: "dup", runFn: func(context.Context, string) (string, error) { return "", nil }})
	if _, err := r.BuildEinoTools(); err == nil {
		t.Fatalf("duplicate names should fail BuildEinoTools")
	}
}
