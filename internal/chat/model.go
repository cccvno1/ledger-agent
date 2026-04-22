package chat

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

// DraftEntry is a single shipment item pending user confirmation.
type DraftEntry struct {
	CustomerID   string    `json:"customer_id"`
	CustomerName string    `json:"customer_name"`
	ProductID    string    `json:"product_id,omitempty"`
	ProductName  string    `json:"product_name"`
	UnitPrice    float64   `json:"unit_price"`
	Quantity     float64   `json:"quantity"`
	Unit         string    `json:"unit"`
	Amount       float64   `json:"amount"`
	EntryDate    time.Time `json:"entry_date"`
	Notes        string    `json:"notes,omitempty"`
	// IdempotencyKey is generated when the entry is added to the draft and
	// passed to LedgerWriter.Create on commit. A retried confirm_draft after
	// a partial failure will re-encounter the same key and the writer will
	// return the surviving row instead of inserting a duplicate.
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// OpEntry records one successful mutation for the structured context block.
type OpEntry struct {
	Time    time.Time         `json:"time"`
	Action  string            `json:"action"`  // "save" | "update" | "delete" | "payment" | "settle" | "query"
	Summary string            `json:"summary"` // human-readable one-liner
	Refs    map[string]string `json:"refs"`    // entity references: "customer_id", "entry_id", "payment_id", etc.
}

// SessionStats tracks aggregate operation counts per session.
type SessionStats struct {
	Saved    int `json:"saved"`
	Modified int `json:"modified"`
	Deleted  int `json:"deleted"`
	Queried  int `json:"queried"`
	Payments int `json:"payments"`
}

const maxOpLog = 10

// Session holds per-conversation state.
type Session struct {
	mu       sync.Mutex // guards all mutable fields below
	ID       string
	Messages []*schema.Message
	Draft    []DraftEntry
	OpLog    []OpEntry
	Stats    SessionStats
	// Phase is the conversational state machine position. Drives the policy
	// layer: tools declare AllowedPhases() and the Registry rejects calls
	// whose tool is not valid in the current phase. Empty string is treated
	// as PhaseIdle (back-compat for sessions persisted before Phase B).
	Phase        Phase
	CreatedAt    time.Time
	LastActiveAt time.Time
}

// CurrentPhase returns Phase, normalising the empty zero value to PhaseIdle.
func (s *Session) CurrentPhase() Phase {
	if s.Phase == "" {
		return PhaseIdle
	}
	return s.Phase
}

// SetPhase mutates the session phase. Callers should hold s.mu (Service.Chat
// already locks it for the entire turn) or invoke from within a tool handler.
func (s *Session) SetPhase(p Phase) {
	s.Phase = p
}

// AppendOp records a successful mutation and updates stats.
func (s *Session) AppendOp(action, summary string, refs map[string]string) {
	entry := OpEntry{
		Time:    time.Now(),
		Action:  action,
		Summary: summary,
		Refs:    refs,
	}
	s.OpLog = append(s.OpLog, entry)
	if len(s.OpLog) > maxOpLog {
		s.OpLog = s.OpLog[len(s.OpLog)-maxOpLog:]
	}

	switch action {
	case "save":
		s.Stats.Saved++
	case "update":
		s.Stats.Modified++
	case "delete":
		s.Stats.Deleted++
	case "query":
		s.Stats.Queried++
	case "payment":
		s.Stats.Payments++
	}
}

// RenderContextBlock produces the structured context block for system prompt injection.
func (s *Session) RenderContextBlock() string {
	var b strings.Builder

	// Draft snapshot
	if len(s.Draft) > 0 {
		b.WriteString("## 当前草稿\n")
		b.WriteString("| 序号 | 客户 | 商品 | 单价 | 数量 | 单位 | 金额 | 日期 |\n")
		b.WriteString("|------|------|------|------|------|------|------|------|\n")
		for i, d := range s.Draft {
			fmt.Fprintf(&b, "| %d | %s | %s | %.2f | %.2f | %s | %.2f | %s |\n",
				i, d.CustomerName, d.ProductName, d.UnitPrice, d.Quantity, d.Unit, d.Amount, d.EntryDate.Format("2006-01-02"))
		}
		b.WriteByte('\n')
	}

	// Operation log
	if len(s.OpLog) > 0 {
		b.WriteString("## 操作记录\n")
		for _, op := range s.OpLog {
			fmt.Fprintf(&b, "- %s %s", op.Time.Format("15:04"), op.Summary)
			if id, ok := op.Refs["entry_id"]; ok {
				fmt.Fprintf(&b, " [entry_id:%s]", id)
			}
			if id, ok := op.Refs["payment_id"]; ok {
				fmt.Fprintf(&b, " [payment_id:%s]", id)
			}
			if id, ok := op.Refs["customer_id"]; ok {
				fmt.Fprintf(&b, " [customer_id:%s]", id)
			}
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}

	// Session stats
	st := s.Stats
	if st.Saved+st.Modified+st.Deleted+st.Queried+st.Payments > 0 {
		fmt.Fprintf(&b, "## 会话统计\n保存%d条 修改%d条 删除%d条 查询%d次 收款%d笔\n",
			st.Saved, st.Modified, st.Deleted, st.Queried, st.Payments)
	}

	return b.String()
}
