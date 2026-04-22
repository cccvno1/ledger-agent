package chat

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

// TestRegistry_PhaseEnforcement verifies that AllowedPhases is structurally
// enforced: a tool restricted to PhaseDrafting cannot run when the live
// session is PhaseIdle.
func TestRegistry_PhaseEnforcement(t *testing.T) {
	store := NewSessionStore()
	sess := store.GetOrCreate("s1")
	sess.SetPhase(PhaseIdle)
	store.Set(sess)

	r := NewRegistry(silentLogger()).WithSessions(store)
	r.Register(&fakeTool{
		name:   "draft_only",
		cap:    CapMutate,
		phases: []Phase{PhaseDrafting}, // not allowed in PhaseIdle
		runFn: func(_ context.Context, _ string) (string, error) {
			t.Fatal("inner tool must not run when phase gate trips")
			return "", nil
		},
	})

	tools, err := r.BuildEinoTools()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	inv := tools[0].(tool.InvokableTool)

	ctx := context.WithValue(context.Background(), sessionIDKey{}, "s1")
	out, err := inv.InvokableRun(ctx, "")
	if err != nil {
		t.Fatalf("wrapper must always return nil err: %v", err)
	}
	if !strings.Contains(out, `"code":"phase_mismatch"`) {
		t.Fatalf("expected phase_mismatch envelope, got: %s", out)
	}
	if !strings.Contains(out, `"current_phase":"idle"`) {
		t.Fatalf("expected current_phase context, got: %s", out)
	}
}

// TestRegistry_PhaseAllowedBypass verifies that tools with empty AllowedPhases
// can run regardless of the session phase.
func TestRegistry_PhaseAllowedBypass(t *testing.T) {
	store := NewSessionStore()
	sess := store.GetOrCreate("s1")
	sess.SetPhase(PhasePendingDestructive)
	store.Set(sess)

	called := false
	r := NewRegistry(silentLogger()).WithSessions(store)
	r.Register(&fakeTool{
		name:   "any_phase",
		cap:    CapRead,
		phases: nil, // any phase
		runFn: func(_ context.Context, _ string) (string, error) {
			called = true
			return `{"ok":true}`, nil
		},
	})

	tools, err := r.BuildEinoTools()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	inv := tools[0].(tool.InvokableTool)

	ctx := context.WithValue(context.Background(), sessionIDKey{}, "s1")
	if _, err := inv.InvokableRun(ctx, ""); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatal("inner tool must run when AllowedPhases is empty")
	}
}

// TestRegistry_PhaseGateBypassedWithoutSession verifies that a Registry
// without a SessionStorer (unit-test mode) does not gate based on phase.
func TestRegistry_PhaseGateBypassedWithoutSession(t *testing.T) {
	called := false
	r := NewRegistry(silentLogger()) // no WithSessions
	r.Register(&fakeTool{
		name:   "draft_only",
		cap:    CapMutate,
		phases: []Phase{PhaseDrafting},
		runFn: func(_ context.Context, _ string) (string, error) {
			called = true
			return `{}`, nil
		},
	})
	tools, _ := r.BuildEinoTools()
	if _, err := tools[0].(tool.InvokableTool).InvokableRun(context.Background(), ""); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatal("phase gate should be a no-op without a session store")
	}
}
