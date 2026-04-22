package chat

// policy.go centralises the Phase × Capability authorisation matrix. Today
// the only enforcement is "tool's AllowedPhases must contain the current
// session phase". As the protocol grows (operation tokens, multi-step
// commits) all gating logic should land here so it stays auditable in one
// file rather than scattered across tool handlers.

// allowsPhase reports whether the tool may run in the given phase. An empty
// allowed set means "any phase".
func allowsPhase(allowed []Phase, current Phase) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, p := range allowed {
		if p == current {
			return true
		}
	}
	return false
}

// phaseSet renders allowed phases for human-readable error hints.
func phaseSet(allowed []Phase) []string {
	out := make([]string, 0, len(allowed))
	for _, p := range allowed {
		out = append(out, string(p))
	}
	return out
}
