package chat

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResolveRefs_LastSaved(t *testing.T) {
	sess := &Session{ID: "s1"}
	sess.AppendOp("save", "saved 2", map[string]string{"entry_ids": "e-old1,e-old2"})
	sess.AppendOp("save", "saved 3", map[string]string{"entry_ids": "e-new1,e-new2,e-new3"})

	cases := map[string]string{
		"last_saved":   "e-new3", // newest within latest batch
		"last_saved.0": "e-new3",
		"last_saved.1": "e-new2",
		"last_saved.2": "e-new1",
		"last_saved.3": "e-old2", // crosses into prior batch
		"last_saved.4": "e-old1",
	}
	for ref, want := range cases {
		args := `{"entry_id":"` + ref + `"}`
		out, err := resolveRefs(args, sess)
		if err != nil {
			t.Fatalf("ref %q unexpected error: %v", ref, err)
		}
		var got map[string]string
		if jerr := json.Unmarshal([]byte(out), &got); jerr != nil {
			t.Fatalf("ref %q: result not JSON: %v (%s)", ref, jerr, out)
		}
		if got["entry_id"] != want {
			t.Errorf("ref %q: got entry_id=%q, want %q", ref, got["entry_id"], want)
		}
	}
}

func TestResolveRefs_OutOfRange(t *testing.T) {
	sess := &Session{ID: "s1"}
	sess.AppendOp("save", "saved 1", map[string]string{"entry_ids": "e1"})

	_, err := resolveRefs(`{"entry_id":"last_saved.5"}`, sess)
	te := AsToolError(err)
	if te == nil {
		t.Fatalf("expected ToolError, got %v", err)
	}
	if te.Code != CodeRefNotFound {
		t.Errorf("code = %s, want %s", te.Code, CodeRefNotFound)
	}
}

func TestResolveRefs_LastPayment(t *testing.T) {
	sess := &Session{ID: "s1"}
	sess.AppendOp("payment", "paid", map[string]string{"payment_id": "p-42", "customer_id": "c-1"})

	out, err := resolveRefs(`{"payment_id":"last_payment"}`, sess)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(out, `"p-42"`) {
		t.Errorf("expected p-42 in %s", out)
	}
}

func TestResolveRefs_PassthroughForNonRef(t *testing.T) {
	sess := &Session{ID: "s1"}
	args := `{"entry_id":"550e8400-e29b-41d4-a716-446655440000","amount":10}`
	out, err := resolveRefs(args, sess)
	if err != nil {
		t.Fatal(err)
	}
	if out != args {
		t.Errorf("non-ref args were rewritten: %s -> %s", args, out)
	}
}

func TestResolveRefs_NilSessionPassthrough(t *testing.T) {
	args := `{"entry_id":"last_saved"}`
	out, err := resolveRefs(args, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != args {
		t.Errorf("nil-session resolveRefs altered args: %s", out)
	}
}
