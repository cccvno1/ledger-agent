package chat

import (
	"strings"
	"testing"
	"time"
)

func TestAppendOp(t *testing.T) {
	sess := &Session{ID: "test"}

	sess.AppendOp("save", "saved 3 entries", map[string]string{"entry_ids": "a,b,c"})

	if len(sess.OpLog) != 1 {
		t.Fatalf("OpLog len = %d, want 1", len(sess.OpLog))
	}
	if sess.OpLog[0].Action != "save" {
		t.Errorf("Action = %q, want %q", sess.OpLog[0].Action, "save")
	}
	if sess.Stats.Saved != 1 {
		t.Errorf("Stats.Saved = %d, want 1", sess.Stats.Saved)
	}
}

func TestAppendOp_Cap(t *testing.T) {
	sess := &Session{ID: "test"}

	for i := 0; i < 15; i++ {
		sess.AppendOp("query", "q", nil)
	}

	if len(sess.OpLog) != maxOpLog {
		t.Fatalf("OpLog len = %d, want %d", len(sess.OpLog), maxOpLog)
	}
	if sess.Stats.Queried != 15 {
		t.Errorf("Stats.Queried = %d, want 15", sess.Stats.Queried)
	}
}

func TestAppendOp_StatsByAction(t *testing.T) {
	sess := &Session{ID: "test"}

	sess.AppendOp("save", "s", nil)
	sess.AppendOp("save", "s", nil)
	sess.AppendOp("update", "u", nil)
	sess.AppendOp("delete", "d", nil)
	sess.AppendOp("query", "q", nil)
	sess.AppendOp("payment", "p", nil)

	if sess.Stats.Saved != 2 {
		t.Errorf("Saved = %d, want 2", sess.Stats.Saved)
	}
	if sess.Stats.Modified != 1 {
		t.Errorf("Modified = %d, want 1", sess.Stats.Modified)
	}
	if sess.Stats.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", sess.Stats.Deleted)
	}
	if sess.Stats.Queried != 1 {
		t.Errorf("Queried = %d, want 1", sess.Stats.Queried)
	}
	if sess.Stats.Payments != 1 {
		t.Errorf("Payments = %d, want 1", sess.Stats.Payments)
	}
}

func TestRenderContextBlock_Empty(t *testing.T) {
	sess := &Session{ID: "test"}
	out := sess.RenderContextBlock()
	if out != "" {
		t.Errorf("empty session should render empty string, got:\n%s", out)
	}
}

func TestRenderContextBlock_Draft(t *testing.T) {
	sess := &Session{
		ID: "test",
		Draft: []DraftEntry{
			{CustomerName: "张三", ProductName: "苹果", UnitPrice: 5.0, Quantity: 10, Unit: "斤", Amount: 50.0, EntryDate: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)},
		},
	}

	out := sess.RenderContextBlock()

	if !strings.Contains(out, "## 当前草稿") {
		t.Error("missing draft header")
	}
	if !strings.Contains(out, "张三") {
		t.Error("missing customer name in draft")
	}
	if !strings.Contains(out, "苹果") {
		t.Error("missing product name in draft")
	}
}

func TestRenderContextBlock_OpLog(t *testing.T) {
	sess := &Session{ID: "test"}
	sess.AppendOp("save", "保存3条记录给张三", map[string]string{"entry_ids": "a,b,c"})
	sess.AppendOp("update", "修改记录", map[string]string{"entry_id": "a"})

	out := sess.RenderContextBlock()

	if !strings.Contains(out, "## 操作记录") {
		t.Error("missing op log header")
	}
	if !strings.Contains(out, "保存3条记录给张三") {
		t.Error("missing save summary")
	}
	if !strings.Contains(out, "[entry_id:a]") {
		t.Error("missing entry_id ref")
	}
}

func TestRenderContextBlock_Stats(t *testing.T) {
	sess := &Session{ID: "test"}
	sess.AppendOp("save", "s", nil)
	sess.AppendOp("query", "q", nil)

	out := sess.RenderContextBlock()

	if !strings.Contains(out, "## 会话统计") {
		t.Error("missing stats header")
	}
	if !strings.Contains(out, "保存1条") {
		t.Error("missing saved count in stats")
	}
	if !strings.Contains(out, "查询1次") {
		t.Error("missing queried count in stats")
	}
}
