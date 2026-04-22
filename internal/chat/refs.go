package chat

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Context references let the LLM target recently-touched entities by their
// position in the session's OpLog instead of memorising opaque UUIDs. The
// resolver inspects every string field in a tool's JSON arguments and, when
// a value matches a recognised ref pattern, rewrites it to the underlying ID
// before the inner tool ever sees it.
//
// Supported syntax:
//   last_saved              → most recent saved entry's id
//   last_saved.N            → the (N+1)-th most recent saved entry (0-indexed from latest)
//   last_payment            → most recent payment id
//
// Resolution failures surface as CodeRefNotFound so the LLM can recover by
// calling search/query tools.

// resolveRefs walks argsJSON and replaces every recognised ref string with
// its concrete id. Returns the (possibly unchanged) JSON. If any ref cannot
// be resolved, returns a *ToolError.
func resolveRefs(argsJSON string, sess *Session) (string, error) {
	if sess == nil || argsJSON == "" {
		return argsJSON, nil
	}
	var parsed any
	if err := json.Unmarshal([]byte(argsJSON), &parsed); err != nil {
		// Not our problem — the inner tool will fail on its own parse and
		// produce a more specific error.
		return argsJSON, nil
	}
	walked, changed, err := walkRefs(parsed, sess)
	if err != nil {
		return "", err
	}
	if !changed {
		return argsJSON, nil
	}
	out, err := json.Marshal(walked)
	if err != nil {
		return argsJSON, nil
	}
	return string(out), nil
}

func walkRefs(v any, sess *Session) (any, bool, error) {
	switch t := v.(type) {
	case map[string]any:
		changed := false
		for k, val := range t {
			nv, c, err := walkRefs(val, sess)
			if err != nil {
				return nil, false, err
			}
			if c {
				t[k] = nv
				changed = true
			}
		}
		return t, changed, nil
	case []any:
		changed := false
		for i, val := range t {
			nv, c, err := walkRefs(val, sess)
			if err != nil {
				return nil, false, err
			}
			if c {
				t[i] = nv
				changed = true
			}
		}
		return t, changed, nil
	case string:
		if !looksLikeRef(t) {
			return v, false, nil
		}
		resolved, err := lookupRef(t, sess)
		if err != nil {
			return nil, false, err
		}
		return resolved, true, nil
	default:
		return v, false, nil
	}
}

func looksLikeRef(s string) bool {
	return strings.HasPrefix(s, "last_saved") || s == "last_payment"
}

func lookupRef(ref string, sess *Session) (string, error) {
	switch {
	case ref == "last_payment":
		for i := len(sess.OpLog) - 1; i >= 0; i-- {
			if sess.OpLog[i].Action == "payment" {
				if id := sess.OpLog[i].Refs["payment_id"]; id != "" {
					return id, nil
				}
			}
		}
		return "", NewToolError(CodeRefNotFound, "no recent payment in session").
			WithHint("call query tools first to obtain a payment_id")

	case ref == "last_saved" || strings.HasPrefix(ref, "last_saved."):
		idx := 0
		if strings.HasPrefix(ref, "last_saved.") {
			n, err := strconv.Atoi(strings.TrimPrefix(ref, "last_saved."))
			if err != nil || n < 0 {
				return "", NewToolError(CodeInvalidArgs,
					fmt.Sprintf("invalid ref %q: expected last_saved or last_saved.N", ref))
			}
			idx = n
		}
		// Collect the most recent save op's entry_ids (comma-separated)
		// then the previous save op's, etc., flatten newest-first.
		ids := collectSavedIDs(sess)
		if idx >= len(ids) {
			return "", NewToolError(CodeRefNotFound,
				fmt.Sprintf("ref %q out of range: only %d saved entries in session", ref, len(ids))).
				WithHint("use query_entries to look up older entries")
		}
		return ids[idx], nil
	}
	return ref, nil
}

// collectSavedIDs walks OpLog from newest to oldest, expanding any "save"
// op's comma-separated entry_ids list. The first element is the most recent
// individual entry id.
func collectSavedIDs(sess *Session) []string {
	var ids []string
	for i := len(sess.OpLog) - 1; i >= 0; i-- {
		op := sess.OpLog[i]
		if op.Action != "save" {
			continue
		}
		raw := op.Refs["entry_ids"]
		if raw == "" {
			continue
		}
		parts := strings.Split(raw, ",")
		// reverse so within a single batch, the "latest" is first
		for j := len(parts) - 1; j >= 0; j-- {
			id := strings.TrimSpace(parts[j])
			if id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}
