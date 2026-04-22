package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Capability is the privilege level required to invoke a tool. Phase B uses
// it together with Phase to authorise calls; in Phase A it is recorded for
// observability only.
type Capability int

const (
	// CapRead — pure reads with no side effects.
	CapRead Capability = iota
	// CapMutate — staging-area writes (draft only); always reversible.
	CapMutate
	// CapCommit — promotes staged data to durable storage.
	CapCommit
	// CapDestructive — irreversible writes; require token protocol.
	CapDestructive
)

func (c Capability) String() string {
	switch c {
	case CapRead:
		return "read"
	case CapMutate:
		return "mutate"
	case CapCommit:
		return "commit"
	case CapDestructive:
		return "destructive"
	default:
		return "unknown"
	}
}

// Phase represents the conversational state of a session. Reserved for
// Phase B; defined here so capability/phase metadata can be co-located now.
type Phase string

const (
	PhaseIdle               Phase = "idle"
	PhaseDrafting           Phase = "drafting"
	PhasePendingCommit      Phase = "pending_commit"
	PhasePendingDestructive Phase = "pending_destructive"
	PhaseCommitted          Phase = "committed"
)

// ManagedTool is the contract that all chat tools implement. It extends
// eino's tool.InvokableTool with metadata the Registry uses for cross-cutting
// concerns (timeouts, audit, future policy enforcement).
type ManagedTool interface {
	tool.InvokableTool

	// Capability declares the privilege level required to invoke this tool.
	Capability() Capability

	// AllowedPhases lists the phases in which the tool may be invoked.
	// An empty slice means "any phase" (used for read-only tools).
	AllowedPhases() []Phase

	// Timeout returns the per-invocation execution budget. Zero means use
	// the Registry's default.
	Timeout() time.Duration
}

// defaultToolTimeout caps any single tool call; can be overridden per tool.
const defaultToolTimeout = 15 * time.Second

// Registry owns the lifecycle of all tools and applies cross-cutting wrappers
// (timeout, panic recovery, error normalisation, audit log) before exposing
// them to the planner. This is the single chokepoint for tool execution.
type Registry struct {
	logger   *slog.Logger
	tools    []ManagedTool
	sessions SessionStorer // optional; required for phase enforcement
}

// NewRegistry creates a Registry with a slog logger for audit output.
func NewRegistry(logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.Default()
	}
	return &Registry{logger: logger}
}

// WithSessions wires a session store so the Registry can resolve the current
// phase before invoking each tool. If unset (e.g. in unit tests with no
// real session machinery), the phase gate is bypassed.
func (r *Registry) WithSessions(s SessionStorer) *Registry {
	r.sessions = s
	return r
}

// currentPhase returns the live phase for the session bound to ctx, or
// (PhaseIdle, nil) when no session is available — signalling the wrapper
// to skip the phase check.
func (r *Registry) currentPhase(ctx context.Context) (Phase, *Session) {
	if r.sessions == nil {
		return PhaseIdle, nil
	}
	id := sessionIDFromCtx(ctx)
	if id == "" {
		return PhaseIdle, nil
	}
	sess := r.sessions.Get(id)
	if sess == nil {
		return PhaseIdle, nil
	}
	return sess.CurrentPhase(), sess
}

// Register adds a tool to the registry. Order is preserved; duplicate names
// are detected at BuildEinoTools time.
func (r *Registry) Register(t ManagedTool) {
	r.tools = append(r.tools, t)
}

// BuildEinoTools returns the wrapped tool set ready to plug into eino's
// react.AgentConfig.ToolsConfig. Each tool is wrapped exactly once.
func (r *Registry) BuildEinoTools() ([]tool.BaseTool, error) {
	seen := make(map[string]struct{}, len(r.tools))
	out := make([]tool.BaseTool, 0, len(r.tools))
	for _, t := range r.tools {
		info, err := t.Info(context.Background())
		if err != nil {
			return nil, fmt.Errorf("registry: tool info: %w", err)
		}
		if _, dup := seen[info.Name]; dup {
			return nil, fmt.Errorf("registry: duplicate tool name %q", info.Name)
		}
		seen[info.Name] = struct{}{}
		out = append(out, &wrappedTool{
			inner:    t,
			name:     info.Name,
			logger:   r.logger,
			timeout:  effectiveTimeout(t.Timeout()),
			meta:     toolMeta{cap: t.Capability(), phases: t.AllowedPhases()},
			registry: r,
		})
	}
	return out, nil
}

// Names returns all registered tool names; useful for diagnostics.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.tools))
	for _, t := range r.tools {
		if info, err := t.Info(context.Background()); err == nil {
			out = append(out, info.Name)
		}
	}
	return out
}

func effectiveTimeout(d time.Duration) time.Duration {
	if d <= 0 {
		return defaultToolTimeout
	}
	return d
}

// toolMeta carries metadata used by the wrapper at call time.
type toolMeta struct {
	cap    Capability
	phases []Phase
}

// wrappedTool implements tool.InvokableTool by delegating to the inner
// ManagedTool while applying:
//
//  1. Per-call timeout (derived context).
//  2. Panic recovery — converts panics into CodeInternal errors.
//  3. Error normalisation — non-ToolError errors are wrapped uniformly.
//  4. Audit logging — one structured slog line per invocation.
//
// The wrapper never alters the tool's success payload, preserving backward
// compatibility with tools that already return JSON strings.
type wrappedTool struct {
	inner    ManagedTool
	name     string
	logger   *slog.Logger
	timeout  time.Duration
	meta     toolMeta
	registry *Registry
}

// Info forwards to the underlying tool unchanged.
func (w *wrappedTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return w.inner.Info(ctx)
}

// InvokableRun applies all cross-cutting concerns and dispatches to the
// inner tool. The returned string is either:
//   - the tool's raw success payload, or
//   - a JSON envelope {"error": {...}} for any failure.
//
// The returned error is always nil; ReAct treats any error from a tool as a
// fatal failure, so we instead surface failures as data the LLM can branch
// on. This is the cornerstone of the "structured feedback" mechanism.
func (w *wrappedTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	start := time.Now()

	// Phase gate: read the live session phase and reject calls whose tool
	// declares incompatible AllowedPhases. This is structural — the LLM
	// cannot "convince" itself past it.
	phase, sess := w.registry.currentPhase(ctx)
	if sess != nil && !allowsPhase(w.meta.phases, phase) {
		te := NewToolError(CodePhaseMismatch,
			fmt.Sprintf("tool %q is not allowed in phase %q", w.name, phase)).
			WithHint(fmt.Sprintf("transition into one of: %v", phaseSet(w.meta.phases))).
			WithContext(map[string]any{
				"current_phase":  string(phase),
				"allowed_phases": phaseSet(w.meta.phases),
				"tool":           w.name,
			})
		return w.emitErr(ctx, te, time.Since(start))
	}

	// Resolve any context refs (last_saved.N, last_payment, ...) before the
	// inner tool sees the args. Failures here become CodeRefNotFound /
	// CodeInvalidArgs envelopes, never reaching the inner tool.
	if sess != nil {
		resolved, err := resolveRefs(args, sess)
		if err != nil {
			return w.emitErr(ctx, classify(err, ctx), time.Since(start))
		}
		args = resolved
	}

	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	result, err := w.runWithRecovery(ctx, args, opts...)
	elapsed := time.Since(start)

	if err == nil {
		w.logger.Info("tool.ok",
			"tool", w.name,
			"capability", w.meta.cap.String(),
			"elapsed_ms", elapsed.Milliseconds(),
			"session_id", sessionIDFromCtx(ctx),
		)
		return result, nil
	}

	// Translate the error into a stable envelope. Any tool that already
	// returned *ToolError is honoured; everything else is bucketed.
	te := classify(err, ctx)
	return w.emitErr(ctx, te, elapsed)
}

// emitErr serialises a ToolError to JSON, logs it, and returns. Centralised
// so the phase gate and the runtime path stay symmetric.
func (w *wrappedTool) emitErr(ctx context.Context, te *ToolError, elapsed time.Duration) (string, error) {
	w.logger.Warn("tool.err",
		"tool", w.name,
		"capability", w.meta.cap.String(),
		"code", string(te.Code),
		"recoverable", te.Recoverable,
		"message", te.Message,
		"hint", te.Hint,
		"elapsed_ms", elapsed.Milliseconds(),
		"session_id", sessionIDFromCtx(ctx),
	)
	envelope, jerr := json.Marshal(te)
	if jerr != nil {
		return fmt.Sprintf(`{"error":{"code":"internal_error","message":%q}}`, te.Message), nil
	}
	return string(envelope), nil
}

// runWithRecovery defends against panics in tool handlers.
func (w *wrappedTool) runWithRecovery(ctx context.Context, args string, opts ...tool.Option) (out string, err error) {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("tool.panic",
				"tool", w.name,
				"recover", fmt.Sprintf("%v", r),
				"stack", string(debug.Stack()),
			)
			err = NewToolError(CodeInternal, "tool panicked").
				WithRecoverable(false).
				WithContext(map[string]any{"panic": fmt.Sprintf("%v", r)})
		}
	}()
	return w.inner.InvokableRun(ctx, args, opts...)
}

// classify turns an arbitrary error into a *ToolError. Honours pre-built
// ToolErrors, errkit-coded domain errors, ctx deadlines, and falls back to
// CodeInternal.
func classify(err error, ctx context.Context) *ToolError {
	if te := AsToolError(err); te != nil {
		return te
	}
	if errors.Is(err, context.DeadlineExceeded) || (ctx != nil && ctx.Err() == context.DeadlineExceeded) {
		return NewToolError(CodeTimeout, "tool execution exceeded its time budget").
			WithHint("retry with simpler arguments or split the request").
			WithRecoverable(true)
	}
	if errors.Is(err, context.Canceled) {
		return NewToolError(CodeInternal, "request was cancelled").WithRecoverable(false)
	}
	if te := FromDomainError(err); te != nil {
		return te
	}
	return NewToolError(CodeInternal, err.Error()).WithRecoverable(false)
}
