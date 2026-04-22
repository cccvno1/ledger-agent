package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

// CustomerSearcher is satisfied by an adapter wrapping customer.Service.
type CustomerSearcher interface {
	// Search returns fuzzy-matched candidates sorted by Levenshtein distance.
	Search(ctx context.Context, query string, topN int) ([]CustomerMatch, error)
	// ListAll returns all customers.
	ListAll(ctx context.Context) ([]CustomerRef, error)
	// Create finds or creates a customer by name and returns its ID and name.
	Create(ctx context.Context, name string) (CustomerRef, error)
	// AddAlias appends an alias to a customer.
	AddAlias(ctx context.Context, in CustomerAliasInput) error
}

// LedgerWriter is satisfied by an adapter wrapping ledger.Service.
type LedgerWriter interface {
	// Create inserts a new ledger entry.
	Create(ctx context.Context, in LedgerCreateInput) (LedgerEntryRef, error)
	// Update modifies an existing ledger entry.
	Update(ctx context.Context, in LedgerUpdateInput) (LedgerEntryRef, error)
	// Delete removes a ledger entry by ID.
	Delete(ctx context.Context, entryID string) error
	// SettleByCustomer marks all unsettled entries for a customer as settled.
	SettleByCustomer(ctx context.Context, customerID string) error
}

// LedgerQuerier is satisfied by an adapter wrapping ledger.Service.
type LedgerQuerier interface {
	// List returns entries matching the filter.
	List(ctx context.Context, in LedgerListInput) ([]LedgerEntryRef, error)
	// SummaryByCustomer returns outstanding totals.
	SummaryByCustomer(ctx context.Context, customerID string) ([]LedgerSummaryRef, error)
	// GetByID returns a single entry; required by propose_delete_entry to
	// build a human-readable preview.
	GetByID(ctx context.Context, id string) (LedgerEntryRef, error)
}

// ProductSearcher is satisfied by an adapter wrapping product.Service.
type ProductSearcher interface {
	Search(ctx context.Context, query string, topN int) ([]ProductMatch, error)
	FindOrCreate(ctx context.Context, name string) (ProductRef, error)
	AddAlias(ctx context.Context, in ProductAliasInput) error
	// ListAll returns the catalogue. Order is service-defined (typically by
	// usage / name).
	ListAll(ctx context.Context) ([]ProductRef, error)
}

// PaymentRecorder is satisfied by an adapter wrapping payment.Service.
type PaymentRecorder interface {
	Create(ctx context.Context, in PaymentCreateInput) (PaymentRef, error)
	TotalByCustomer(ctx context.Context, customerID string) (float64, error)
	// ListByCustomer returns all payments for a customer, newest first.
	ListByCustomer(ctx context.Context, customerID string) ([]PaymentRef, error)
}

// --- Shared DTO types for the cross-feature boundary ---

// CustomerMatch is a fuzzy-search result.
type CustomerMatch struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Score      int    `json:"score"`
	MatchedVia string `json:"matched_via"` // "exact", "alias", "fuzzy"
}

// CustomerRef is a minimal customer identifier.
type CustomerRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CustomerAliasInput carries alias data for a customer.
type CustomerAliasInput struct {
	CustomerID string
	Alias      string
}

// ProductMatch is a product fuzzy-search result.
type ProductMatch struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	DefaultUnit    string  `json:"default_unit"`
	ReferencePrice float64 `json:"reference_price,omitempty"`
	Score          int     `json:"score"`
	MatchedVia     string  `json:"matched_via"` // "exact", "alias", "fuzzy"
}

// ProductRef is a minimal product identifier.
type ProductRef struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	DefaultUnit    string  `json:"default_unit"`
	ReferencePrice float64 `json:"reference_price,omitempty"`
}

// ProductAliasInput carries alias data for a product.
type ProductAliasInput struct {
	ProductID string
	Alias     string
}

// PaymentCreateInput carries fields for recording a payment.
type PaymentCreateInput struct {
	CustomerID  string
	Amount      float64
	PaymentDate time.Time
	Notes       string
}

// PaymentRef is a minimal payment result.
type PaymentRef struct {
	ID          string    `json:"id"`
	Amount      float64   `json:"amount"`
	PaymentDate time.Time `json:"payment_date"`
}

// LedgerCreateInput carries fields for a new entry.
type LedgerCreateInput struct {
	CustomerID   string
	CustomerName string
	ProductName  string
	UnitPrice    float64
	Quantity     float64
	Unit         string
	EntryDate    time.Time
	Notes        string
	// IdempotencyKey, when set, makes this Create safe to retry.
	IdempotencyKey string
}

// LedgerUpdateInput carries mutable fields for an existing entry.
type LedgerUpdateInput struct {
	ID           string
	CustomerName string
	ProductName  string
	UnitPrice    float64
	Quantity     float64
	Unit         string
	EntryDate    time.Time
	Notes        string
}

// LedgerListInput carries query filters.
type LedgerListInput struct {
	CustomerID   string
	CustomerName string
	DateFrom     *time.Time
	DateTo       *time.Time
	IsSettled    *bool
}

// LedgerEntryRef is a minimal ledger entry.
type LedgerEntryRef struct {
	ID           string    `json:"id"`
	CustomerName string    `json:"customer_name"`
	ProductName  string    `json:"product_name"`
	UnitPrice    float64   `json:"unit_price"`
	Quantity     float64   `json:"quantity"`
	Unit         string    `json:"unit"`
	Amount       float64   `json:"amount"`
	EntryDate    time.Time `json:"entry_date"`
	IsSettled    bool      `json:"is_settled"`
	Notes        string    `json:"notes,omitempty"`
}

// LedgerSummaryRef is an aggregated customer balance.
type LedgerSummaryRef struct {
	CustomerID    string  `json:"customer_id"`
	CustomerName  string  `json:"customer_name"`
	TotalAmount   float64 `json:"total_amount"`
	PendingAmount float64 `json:"pending_amount"`
	EntryCount    int     `json:"entry_count"`
}

// --- Tool builders ---

// buildRegistry constructs the agent's tool Registry. The Registry owns
// cross-cutting concerns (timeout, panic recovery, error normalisation,
// audit log) so individual tools can stay focused on business logic.
func buildRegistry(logger *slog.Logger, sessions SessionStorer, searcher CustomerSearcher, writer LedgerWriter, querier LedgerQuerier, products ProductSearcher, payments PaymentRecorder) *Registry {
	r := NewRegistry(logger).WithSessions(sessions)
	tokens := NewTokenStore()
	r.Register(&searchCustomerTool{sessions: sessions, searcher: searcher})
	r.Register(&listCustomersTool{sessions: sessions, searcher: searcher})
	r.Register(&addToDraftTool{sessions: sessions, searcher: searcher, productSearcher: products})
	r.Register(&updateDraftItemTool{sessions: sessions})
	r.Register(&removeDraftItemTool{sessions: sessions})
	r.Register(&clearDraftTool{sessions: sessions})
	r.Register(&confirmDraftTool{sessions: sessions, writer: writer, searcher: searcher, productSearcher: products})
	r.Register(&queryEntriesTool{sessions: sessions, querier: querier})
	r.Register(&updateEntryTool{sessions: sessions, writer: writer})
	r.Register(&proposeDeleteEntryTool{sessions: sessions, querier: querier, tokens: tokens})
	r.Register(&proposeSettleAccountTool{sessions: sessions, querier: querier, tokens: tokens})
	r.Register(&commitOperationTool{sessions: sessions, writer: writer, tokens: tokens})
	r.Register(&calculateSummaryTool{sessions: sessions, querier: querier, payments: payments})
	r.Register(&recordPaymentTool{sessions: sessions, searcher: searcher, payments: payments})
	r.Register(&listProductsTool{products: products})
	r.Register(&listPaymentsTool{sessions: sessions, searcher: searcher, payments: payments})
	r.Register(&addCustomerAliasTool{searcher: searcher})
	r.Register(&addProductAliasTool{products: products})
	return r
}

// --- search_customer ---

type searchCustomerTool struct {
	sessions SessionStorer
	searcher CustomerSearcher
}

func (t *searchCustomerTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "search_customer",
		Desc: "模糊搜索客户名称，返回最相似的候选项（Levenshtein距离排序）",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "客户名称关键词", Required: true},
		}),
	}, nil
}

func (t *searchCustomerTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", fmt.Errorf("search_customer: parse args: %w", err)
	}
	results, err := t.searcher.Search(ctx, p.Query, 5)
	if err != nil {
		return "", fmt.Errorf("search_customer: %w", err)
	}
	return mustJSON(results), nil
}

// --- list_customers ---

type listCustomersTool struct {
	sessions SessionStorer
	searcher CustomerSearcher
}

func (t *listCustomersTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "list_customers",
		Desc:        "列出所有客户",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *listCustomersTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	customers, err := t.searcher.ListAll(ctx)
	if err != nil {
		return "", fmt.Errorf("list_customers: %w", err)
	}
	return mustJSON(customers), nil
}

// --- add_to_draft ---

type addToDraftTool struct {
	sessions        SessionStorer
	searcher        CustomerSearcher
	productSearcher ProductSearcher
}

func (t *addToDraftTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "add_to_draft",
		Desc: "将一条出货记录添加到草稿。客户名和商品名传用户原话，系统精确匹配名称或别名，不匹配则新建。日期支持'今天'、'昨天'、'前天'、'N天前'、'上周一'等表达式，系统自动计算。金额由系统自动计算（单价×数量）。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"customer_name": {Type: schema.String, Desc: "客户名称或别名（原样传递用户说的）", Required: true},
			"product_name":  {Type: schema.String, Desc: "商品名称（原样传递用户说的，系统自动匹配商品目录）", Required: true},
			"unit_price":    {Type: schema.Number, Desc: "单价（元），必须原样传递用户给出的数字", Required: true},
			"quantity":      {Type: schema.Number, Desc: "数量，必须原样传递用户给出的数字", Required: true},
			"unit":          {Type: schema.String, Desc: "计量单位（如 斤、箱、个、袋），若不确定可不传，系统会用商品默认单位"},
			"date_expr":     {Type: schema.String, Desc: "日期表达式，支持：YYYY-MM-DD、今天、昨天、前天、大前天、N天前、上周一~上周日", Required: true},
			"notes":         {Type: schema.String, Desc: "备注"},
		}),
	}, nil
}

func (t *addToDraftTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		CustomerName string  `json:"customer_name"`
		ProductName  string  `json:"product_name"`
		UnitPrice    float64 `json:"unit_price"`
		Quantity     float64 `json:"quantity"`
		Unit         string  `json:"unit"`
		DateExpr     string  `json:"date_expr"`
		Notes        string  `json:"notes"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", fmt.Errorf("add_to_draft: parse args: %w", err)
	}

	// 1. Parse date expression server-side
	if p.DateExpr == "" {
		return "", fmt.Errorf("add_to_draft: date_expr is required")
	}
	entryDate, err := parseDateExpr(p.DateExpr, time.Now())
	if err != nil {
		return "", fmt.Errorf("add_to_draft: invalid date_expr %q: %w", p.DateExpr, err)
	}

	resolutions := make(map[string]map[string]string)

	// 2. Match product: exact name or alias only, otherwise create new.
	var productID, productName, unit string
	prodRef, err := t.productSearcher.FindOrCreate(ctx, p.ProductName)
	if err != nil {
		return "", fmt.Errorf("add_to_draft: resolve product: %w", err)
	}
	productID = prodRef.ID
	productName = prodRef.Name
	unit = prodRef.DefaultUnit
	if p.Unit != "" {
		unit = p.Unit
	}
	resolutions["product"] = map[string]string{
		"input":   p.ProductName,
		"matched": prodRef.Name,
	}

	// 3. Match customer: exact name or alias only, otherwise create new.
	var customerID, customerName string
	custRef, err := t.searcher.Create(ctx, p.CustomerName)
	if err != nil {
		return "", fmt.Errorf("add_to_draft: resolve customer: %w", err)
	}
	customerID = custRef.ID
	customerName = custRef.Name
	resolutions["customer"] = map[string]string{
		"input":   p.CustomerName,
		"matched": custRef.Name,
	}

	// 4. Compute amount
	amount := p.UnitPrice * p.Quantity

	// 5. Add to draft
	sess, unlock := lockedSession(ctx, t.sessions)
	defer unlock()
	sess.Draft = append(sess.Draft, DraftEntry{
		CustomerID:     customerID,
		CustomerName:   customerName,
		ProductID:      productID,
		ProductName:    productName,
		UnitPrice:      p.UnitPrice,
		Quantity:       p.Quantity,
		Unit:           unit,
		Amount:         amount,
		EntryDate:      entryDate,
		Notes:          p.Notes,
		IdempotencyKey: uuid.NewString(),
	})
	sess.SetPhase(PhaseDrafting)
	t.sessions.Set(sess)

	return mustJSON(map[string]any{
		"draft":       sess.Draft,
		"resolutions": resolutions,
		"message": fmt.Sprintf("已添加: %s %s %.2f%s 单价%.2f 金额%.2f 日期%s，当前草稿共 %d 条",
			customerName, productName, p.Quantity, unit, p.UnitPrice, amount, entryDate.Format("2006-01-02"), len(sess.Draft)),
	}), nil
}

// --- update_draft_item ---

type updateDraftItemTool struct {
	sessions SessionStorer
}

func (t *updateDraftItemTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "update_draft_item",
		Desc: "修改草稿中指定序号的条目（序号从0开始）。修改单价或数量后金额会自动重新计算。日期支持表达式。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"index":        {Type: schema.Integer, Desc: "草稿条目序号（从0开始）", Required: true},
			"product_name": {Type: schema.String, Desc: "新商品名称"},
			"unit_price":   {Type: schema.Number, Desc: "新单价"},
			"quantity":     {Type: schema.Number, Desc: "新数量"},
			"unit":         {Type: schema.String, Desc: "新计量单位"},
			"date_expr":    {Type: schema.String, Desc: "新日期（支持日期表达式）"},
			"notes":        {Type: schema.String, Desc: "新备注"},
		}),
	}, nil
}

func (t *updateDraftItemTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		Index       int     `json:"index"`
		ProductName string  `json:"product_name"`
		UnitPrice   float64 `json:"unit_price"`
		Quantity    float64 `json:"quantity"`
		Unit        string  `json:"unit"`
		DateExpr    string  `json:"date_expr"`
		Notes       string  `json:"notes"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", fmt.Errorf("update_draft_item: parse args: %w", err)
	}

	sess, unlock := lockedSession(ctx, t.sessions)
	defer unlock()
	if p.Index < 0 || p.Index >= len(sess.Draft) {
		return "", fmt.Errorf("update_draft_item: index %d out of range (draft has %d items)", p.Index, len(sess.Draft))
	}

	item := &sess.Draft[p.Index]
	if p.ProductName != "" {
		item.ProductName = p.ProductName
	}
	if p.UnitPrice > 0 {
		item.UnitPrice = p.UnitPrice
	}
	if p.Quantity > 0 {
		item.Quantity = p.Quantity
	}
	if p.Unit != "" {
		item.Unit = p.Unit
	}
	if p.DateExpr != "" {
		d, err := parseDateExpr(p.DateExpr, time.Now())
		if err != nil {
			return "", fmt.Errorf("update_draft_item: invalid date_expr: %w", err)
		}
		item.EntryDate = d
	}
	if p.Notes != "" {
		item.Notes = p.Notes
	}
	item.Amount = item.UnitPrice * item.Quantity
	t.sessions.Set(sess)

	return mustJSON(map[string]any{"draft": sess.Draft}), nil
}

// --- clear_draft ---

type clearDraftTool struct {
	sessions SessionStorer
}

func (t *clearDraftTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "clear_draft",
		Desc:        "清空当前草稿",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *clearDraftTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	sess, unlock := lockedSession(ctx, t.sessions)
	defer unlock()
	sess.Draft = nil
	sess.SetPhase(PhaseIdle)
	t.sessions.Set(sess)
	return `{"message":"草稿已清空"}`, nil
}

// --- remove_draft_item ---

type removeDraftItemTool struct {
	sessions SessionStorer
}

func (t *removeDraftItemTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "remove_draft_item",
		Desc: "删除草稿中指定序号的条目（序号从0开始）",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"index": {Type: schema.Integer, Desc: "要删除的草稿条目序号（从0开始）", Required: true},
		}),
	}, nil
}

func (t *removeDraftItemTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", fmt.Errorf("remove_draft_item: parse args: %w", err)
	}

	sess, unlock := lockedSession(ctx, t.sessions)
	defer unlock()
	if p.Index < 0 || p.Index >= len(sess.Draft) {
		return "", fmt.Errorf("remove_draft_item: index %d out of range (draft has %d items)", p.Index, len(sess.Draft))
	}

	sess.Draft = append(sess.Draft[:p.Index], sess.Draft[p.Index+1:]...)
	if len(sess.Draft) == 0 {
		sess.SetPhase(PhaseIdle)
	}
	t.sessions.Set(sess)

	return mustJSON(map[string]any{
		"draft":   sess.Draft,
		"message": fmt.Sprintf("已删除第 %d 条，当前草稿共 %d 条", p.Index, len(sess.Draft)),
	}), nil
}

// --- confirm_draft ---

type confirmDraftTool struct {
	sessions        SessionStorer
	writer          LedgerWriter
	searcher        CustomerSearcher
	productSearcher ProductSearcher
}

func (t *confirmDraftTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "confirm_draft",
		Desc:        "将草稿中的所有条目保存为正式记录。仅在用户明确确认后调用。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *confirmDraftTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	sess, unlock := lockedSession(ctx, t.sessions)
	defer unlock()
	if len(sess.Draft) == 0 {
		return `{"message":"草稿为空，无需保存"}`, nil
	}

	// Idempotent commit: each draft slot carries its own IdempotencyKey, so
	// re-invoking confirm_draft after a partial failure resolves to the same
	// rows instead of duplicating them. We collect successes and the index of
	// the first failure (if any); on full success we clear the draft, on
	// partial failure we drop the already-saved prefix and return a
	// recoverable ToolError so the LLM can re-confirm without producing dups.
	entryIDs := make([]string, 0, len(sess.Draft))
	for i, d := range sess.Draft {
		ref, err := t.writer.Create(ctx, LedgerCreateInput{
			CustomerID:     d.CustomerID,
			CustomerName:   d.CustomerName,
			ProductName:    d.ProductName,
			UnitPrice:      d.UnitPrice,
			Quantity:       d.Quantity,
			Unit:           d.Unit,
			EntryDate:      d.EntryDate,
			Notes:          d.Notes,
			IdempotencyKey: d.IdempotencyKey,
		})
		if err != nil {
			// Trim already-saved prefix so a retry only attempts the rest.
			sess.Draft = sess.Draft[i:]
			t.sessions.Set(sess)
			te := FromDomainError(err)
			if te == nil {
				te = NewToolError(CodeInternal, err.Error())
			}
			te.WithContext(map[string]any{
				"saved_so_far": entryIDs,
				"failed_index": i,
				"remaining":    len(sess.Draft),
			}).WithHint("草稿已保留未保存部分，可重试 confirm_draft")
			return "", te
		}
		entryIDs = append(entryIDs, ref.ID)
	}
	saved := len(entryIDs)

	results := make([]map[string]any, 0, saved)
	for i, d := range sess.Draft {
		results = append(results, map[string]any{
			"entry_id":      entryIDs[i],
			"customer_name": d.CustomerName,
			"product_name":  d.ProductName,
			"unit_price":    d.UnitPrice,
			"quantity":      d.Quantity,
			"unit":          d.Unit,
			"amount":        d.Amount,
			"entry_date":    d.EntryDate.Format("2006-01-02"),
		})
	}

	// Build summary for operation log
	customers := make(map[string]bool)
	var totalAmount float64
	for _, d := range sess.Draft {
		customers[d.CustomerName] = true
		totalAmount += d.Amount
	}
	names := make([]string, 0, len(customers))
	for n := range customers {
		names = append(names, n)
	}
	sess.AppendOp("save",
		fmt.Sprintf("保存%d条记录给%s，合计%.2f元", saved, strings.Join(names, "、"), totalAmount),
		map[string]string{"entry_ids": strings.Join(entryIDs, ",")})

	sess.Draft = nil
	sess.SetPhase(PhaseCommitted)
	t.sessions.Set(sess)
	return mustJSON(map[string]any{"saved": saved, "entries": results, "entry_ids": entryIDs, "message": fmt.Sprintf("已成功保存 %d 条记录", saved)}), nil
}

// --- query_entries ---

type queryEntriesTool struct {
	sessions SessionStorer
	querier  LedgerQuerier
}

func (t *queryEntriesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "query_entries",
		Desc: "查询历史出货记录，支持按客户名/日期/是否清账过滤。日期参数支持表达式（如'昨天'、'上周一'）。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"customer_id":   {Type: schema.String, Desc: "客户ID（精确）"},
			"customer_name": {Type: schema.String, Desc: "客户名称（模糊）"},
			"date_from":     {Type: schema.String, Desc: "开始日期（支持日期表达式）"},
			"date_to":       {Type: schema.String, Desc: "结束日期（支持日期表达式）"},
			"is_settled":    {Type: schema.Boolean, Desc: "是否已清账"},
		}),
	}, nil
}

func (t *queryEntriesTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		CustomerID   string `json:"customer_id"`
		CustomerName string `json:"customer_name"`
		DateFrom     string `json:"date_from"`
		DateTo       string `json:"date_to"`
		IsSettled    *bool  `json:"is_settled"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", fmt.Errorf("query_entries: parse args: %w", err)
	}

	in := LedgerListInput{
		CustomerID:   p.CustomerID,
		CustomerName: p.CustomerName,
		IsSettled:    p.IsSettled,
	}
	if p.DateFrom != "" {
		d, err := parseDateExpr(p.DateFrom, time.Now())
		if err != nil {
			return "", fmt.Errorf("query_entries: invalid date_from: %w", err)
		}
		in.DateFrom = &d
	}
	if p.DateTo != "" {
		d, err := parseDateExpr(p.DateTo, time.Now())
		if err != nil {
			return "", fmt.Errorf("query_entries: invalid date_to: %w", err)
		}
		in.DateTo = &d
	}

	entries, err := t.querier.List(ctx, in)
	if err != nil {
		return "", fmt.Errorf("query_entries: %w", err)
	}

	sess, unlock := lockedSession(ctx, t.sessions)
	defer unlock()
	summary := fmt.Sprintf("查询到 %d 条记录", len(entries))
	if p.CustomerName != "" {
		summary += "（" + p.CustomerName + "）"
	}
	sess.AppendOp("query", summary, nil)
	t.sessions.Set(sess)

	return mustJSON(entries), nil
}

// --- update_entry ---

type updateEntryTool struct {
	sessions SessionStorer
	writer   LedgerWriter
}

func (t *updateEntryTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "update_entry",
		Desc: "修改一条已保存的出货记录。日期支持表达式。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"entry_id":     {Type: schema.String, Desc: "记录ID", Required: true},
			"product_name": {Type: schema.String, Desc: "新商品名称"},
			"unit_price":   {Type: schema.Number, Desc: "新单价"},
			"quantity":     {Type: schema.Number, Desc: "新数量"},
			"unit":         {Type: schema.String, Desc: "新计量单位"},
			"date_expr":    {Type: schema.String, Desc: "新日期（支持日期表达式）"},
			"notes":        {Type: schema.String, Desc: "新备注"},
		}),
	}, nil
}

func (t *updateEntryTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		EntryID     string  `json:"entry_id"`
		ProductName string  `json:"product_name"`
		UnitPrice   float64 `json:"unit_price"`
		Quantity    float64 `json:"quantity"`
		Unit        string  `json:"unit"`
		DateExpr    string  `json:"date_expr"`
		Notes       string  `json:"notes"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", fmt.Errorf("update_entry: parse args: %w", err)
	}

	in := LedgerUpdateInput{
		ID:          p.EntryID,
		ProductName: p.ProductName,
		UnitPrice:   p.UnitPrice,
		Quantity:    p.Quantity,
		Unit:        p.Unit,
		Notes:       p.Notes,
	}
	if p.DateExpr != "" {
		d, err := parseDateExpr(p.DateExpr, time.Now())
		if err != nil {
			return "", fmt.Errorf("update_entry: invalid date_expr: %w", err)
		}
		in.EntryDate = d
	}

	entry, err := t.writer.Update(ctx, in)
	if err != nil {
		return "", fmt.Errorf("update_entry: %w", err)
	}

	sess, unlock := lockedSession(ctx, t.sessions)
	defer unlock()
	var changes []string
	if p.ProductName != "" {
		changes = append(changes, "商品→"+p.ProductName)
	}
	if p.UnitPrice != 0 {
		changes = append(changes, fmt.Sprintf("单价→%.2f", p.UnitPrice))
	}
	if p.Quantity != 0 {
		changes = append(changes, fmt.Sprintf("数量→%.2f", p.Quantity))
	}
	if p.DateExpr != "" {
		changes = append(changes, "日期→"+p.DateExpr)
	}
	sess.AppendOp("update",
		fmt.Sprintf("修改记录: %s", strings.Join(changes, ", ")),
		map[string]string{"entry_id": p.EntryID})
	t.sessions.Set(sess)

	return mustJSON(entry), nil
}

// --- propose_delete_entry ---
//
// Two-step destructive flow: propose_X stages an operation and returns an
// opaque token + human-readable preview. The agent surfaces the preview to
// the user; only after explicit confirmation should it call commit_operation
// with the token. Tokens are session-scoped and expire after 5 minutes.

type proposeDeleteEntryTool struct {
	sessions SessionStorer
	querier  LedgerQuerier
	tokens   *TokenStore
}

func (t *proposeDeleteEntryTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "propose_delete_entry",
		Desc: "提议删除一条出货记录。返回 operation_token 与 preview，需用户明确同意后再调用 commit_operation。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"entry_id": {Type: schema.String, Desc: "记录ID", Required: true},
		}),
	}, nil
}

func (t *proposeDeleteEntryTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		EntryID string `json:"entry_id"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", NewToolError(CodeInvalidArgs, fmt.Sprintf("propose_delete_entry: parse args: %v", err))
	}
	if strings.TrimSpace(p.EntryID) == "" {
		return "", NewToolError(CodeInvalidArgs, "entry_id is required")
	}

	entry, err := t.querier.GetByID(ctx, p.EntryID)
	if err != nil {
		if te := FromDomainError(err); te != nil {
			return "", te
		}
		return "", fmt.Errorf("propose_delete_entry: %w", err)
	}

	preview := fmt.Sprintf("将删除：%s 于 %s 购买 %s %.2f%s，金额 %.2f 元",
		entry.CustomerName,
		entry.EntryDate.Format("2006-01-02"),
		entry.ProductName, entry.Quantity, entry.Unit, entry.Amount,
	)

	sess, unlock := lockedSession(ctx, t.sessions)
	defer unlock()
	op := t.tokens.Issue(sess.ID, OpDeleteEntry, map[string]any{
		"entry_id": entry.ID,
	}, preview)
	sess.SetPhase(PhasePendingDestructive)
	t.sessions.Set(sess)

	return mustJSON(map[string]any{
		"operation_token": op.Token,
		"kind":            string(op.Kind),
		"preview":         preview,
		"expires_in":      int(tokenTTL.Seconds()),
		"message":         "请向用户读出 preview 并请求明确确认，随后调用 commit_operation 提交",
	}), nil
}

// --- propose_settle_account ---

type proposeSettleAccountTool struct {
	sessions SessionStorer
	querier  LedgerQuerier
	tokens   *TokenStore
}

func (t *proposeSettleAccountTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "propose_settle_account",
		Desc: "提议为指定客户清账。返回 operation_token 与 preview，需用户明确同意后再调用 commit_operation。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"customer_id": {Type: schema.String, Desc: "客户ID", Required: true},
		}),
	}, nil
}

func (t *proposeSettleAccountTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		CustomerID string `json:"customer_id"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", NewToolError(CodeInvalidArgs, fmt.Sprintf("propose_settle_account: parse args: %v", err))
	}
	if strings.TrimSpace(p.CustomerID) == "" {
		return "", NewToolError(CodeInvalidArgs, "customer_id is required")
	}

	summaries, err := t.querier.SummaryByCustomer(ctx, p.CustomerID)
	if err != nil {
		return "", fmt.Errorf("propose_settle_account: summary: %w", err)
	}
	if len(summaries) == 0 {
		return "", NewToolError(CodeNotFound, "客户无可清账记录").WithContext(map[string]any{"customer_id": p.CustomerID})
	}
	s0 := summaries[0]
	preview := fmt.Sprintf("将将 %s 的 %d 条未清账记录全部标为已清，总额 %.2f 元（待付 %.2f）",
		s0.CustomerName, s0.EntryCount, s0.TotalAmount, s0.PendingAmount)

	sess, unlock := lockedSession(ctx, t.sessions)
	defer unlock()
	op := t.tokens.Issue(sess.ID, OpSettleAccount, map[string]any{
		"customer_id":   s0.CustomerID,
		"customer_name": s0.CustomerName,
	}, preview)
	sess.SetPhase(PhasePendingDestructive)
	t.sessions.Set(sess)

	return mustJSON(map[string]any{
		"operation_token": op.Token,
		"kind":            string(op.Kind),
		"preview":         preview,
		"expires_in":      int(tokenTTL.Seconds()),
		"message":         "请向用户读出 preview 并请求明确确认，随后调用 commit_operation 提交",
	}), nil
}

// --- commit_operation ---

type commitOperationTool struct {
	sessions SessionStorer
	writer   LedgerWriter
	tokens   *TokenStore
}

func (t *commitOperationTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "commit_operation",
		Desc: "提交先前通过 propose_* 提议的不可逆操作。仅在用户明确同意后调用。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"operation_token": {Type: schema.String, Desc: "propose_* 返回的 token", Required: true},
		}),
	}, nil
}

func (t *commitOperationTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		OperationToken string `json:"operation_token"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", NewToolError(CodeInvalidArgs, fmt.Sprintf("commit_operation: parse args: %v", err))
	}
	if strings.TrimSpace(p.OperationToken) == "" {
		return "", NewToolError(CodeInvalidArgs, "operation_token is required")
	}

	sess, unlock := lockedSession(ctx, t.sessions)
	defer unlock()
	op := t.tokens.Consume(p.OperationToken, sess.ID)
	if op == nil {
		return "", NewToolError(CodeTokenInvalid, "operation_token 无效、已过期或不属于当前会话").
			WithHint("重新调用 propose_* 获取新 token")
	}

	switch op.Kind {
	case OpDeleteEntry:
		entryID, _ := op.Payload["entry_id"].(string)
		if err := t.writer.Delete(ctx, entryID); err != nil {
			if te := FromDomainError(err); te != nil {
				return "", te
			}
			return "", fmt.Errorf("commit_operation: delete: %w", err)
		}
		sess.AppendOp("delete", op.Preview, map[string]string{"entry_id": entryID})
		sess.SetPhase(PhaseIdle)
		t.sessions.Set(sess)
		return mustJSON(map[string]any{
			"committed":  string(op.Kind),
			"deleted_id": entryID,
			"message":    "记录已删除",
		}), nil

	case OpSettleAccount:
		customerID, _ := op.Payload["customer_id"].(string)
		if err := t.writer.SettleByCustomer(ctx, customerID); err != nil {
			if te := FromDomainError(err); te != nil {
				return "", te
			}
			return "", fmt.Errorf("commit_operation: settle: %w", err)
		}
		sess.AppendOp("settle", op.Preview, map[string]string{"customer_id": customerID})
		sess.SetPhase(PhaseIdle)
		t.sessions.Set(sess)
		return mustJSON(map[string]any{
			"committed":   string(op.Kind),
			"customer_id": customerID,
			"message":     "清账成功",
		}), nil

	default:
		return "", NewToolError(CodeInternal, fmt.Sprintf("unknown operation kind %q", op.Kind))
	}
}

// --- calculate_summary ---

type calculateSummaryTool struct {
	sessions SessionStorer
	querier  LedgerQuerier
	payments PaymentRecorder
}

func (t *calculateSummaryTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "calculate_summary",
		Desc: "汇总指定客户（或所有客户）的应收款总额，包含已收款和实际余额",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"customer_id": {Type: schema.String, Desc: "客户ID（留空则汇总所有客户）"},
		}),
	}, nil
}

func (t *calculateSummaryTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		CustomerID string `json:"customer_id"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", fmt.Errorf("calculate_summary: parse args: %w", err)
	}

	summaries, err := t.querier.SummaryByCustomer(ctx, p.CustomerID)
	if err != nil {
		return "", fmt.Errorf("calculate_summary: %w", err)
	}

	// Enrich with payment info
	type enrichedSummary struct {
		LedgerSummaryRef
		TotalPaid float64 `json:"total_paid"`
		Balance   float64 `json:"balance"` // pending_amount - total_paid
	}
	results := make([]enrichedSummary, 0, len(summaries))
	for _, s := range summaries {
		totalPaid, err := t.payments.TotalByCustomer(ctx, s.CustomerID)
		if err != nil {
			totalPaid = 0 // non-critical, fallback
		}
		results = append(results, enrichedSummary{
			LedgerSummaryRef: s,
			TotalPaid:        totalPaid,
			Balance:          s.PendingAmount - totalPaid,
		})
	}
	return mustJSON(results), nil
}

// --- record_payment ---

type recordPaymentTool struct {
	sessions SessionStorer
	searcher CustomerSearcher
	payments PaymentRecorder
}

func (t *recordPaymentTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "record_payment",
		Desc: "记录客户的一笔收款。客户名需精确匹配，日期支持表达式。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"customer_name": {Type: schema.String, Desc: "客户名称（模糊匹配）", Required: true},
			"amount":        {Type: schema.Number, Desc: "收款金额", Required: true},
			"date_expr":     {Type: schema.String, Desc: "收款日期（支持日期表达式）", Required: true},
			"notes":         {Type: schema.String, Desc: "备注"},
		}),
	}, nil
}

func (t *recordPaymentTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		CustomerName string  `json:"customer_name"`
		Amount       float64 `json:"amount"`
		DateExpr     string  `json:"date_expr"`
		Notes        string  `json:"notes"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", fmt.Errorf("record_payment: parse args: %w", err)
	}

	// Parse date
	payDate, err := parseDateExpr(p.DateExpr, time.Now())
	if err != nil {
		return "", fmt.Errorf("record_payment: invalid date_expr: %w", err)
	}

	// Match customer
	results, err := t.searcher.Search(ctx, p.CustomerName, 3)
	if err != nil {
		return "", fmt.Errorf("record_payment: search customer: %w", err)
	}
	if len(results) == 0 || results[0].Score > 0 {
		return "", NewToolError(CodeNotFound,
			fmt.Sprintf("找不到精确匹配的客户 %q", p.CustomerName)).
			WithHint("调用 search_customer 查看近似候选与用户确认，或使用 add_customer_alias 添加别名").
			WithContext(map[string]any{
				"query":      p.CustomerName,
				"candidates": results,
			})
	}
	customer := results[0]

	// Record payment
	ref, err := t.payments.Create(ctx, PaymentCreateInput{
		CustomerID:  customer.ID,
		Amount:      p.Amount,
		PaymentDate: payDate,
		Notes:       p.Notes,
	})
	if err != nil {
		return "", fmt.Errorf("record_payment: %w", err)
	}

	// Get remaining balance
	totalPaid, _ := t.payments.TotalByCustomer(ctx, customer.ID)

	sess, unlock := lockedSession(ctx, t.sessions)
	defer unlock()
	sess.AppendOp("payment",
		fmt.Sprintf("%s 收款 %.2f 元，累计已收 %.2f 元", customer.Name, p.Amount, totalPaid),
		map[string]string{"payment_id": ref.ID, "customer_id": customer.ID})
	t.sessions.Set(sess)

	return mustJSON(map[string]any{
		"payment_id":    ref.ID,
		"customer_name": customer.Name,
		"amount":        p.Amount,
		"payment_date":  payDate.Format("2006-01-02"),
		"total_paid":    totalPaid,
		"message":       fmt.Sprintf("已记录 %s 收款 %.2f 元（%s），累计已收 %.2f 元", customer.Name, p.Amount, payDate.Format("2006-01-02"), totalPaid),
	}), nil
}

// mustJSON marshals v to JSON, returning an error string on failure.
func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":"json marshal failed: %s"}`, err.Error())
	}
	return string(b)
}

// --- list_products ---

type listProductsTool struct {
	products ProductSearcher
}

func (t *listProductsTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "list_products",
		Desc:        "列出商品目录（含别名、默认单位、参考价）",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *listProductsTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	products, err := t.products.ListAll(ctx)
	if err != nil {
		return "", fmt.Errorf("list_products: %w", err)
	}
	return mustJSON(map[string]any{"products": products, "count": len(products)}), nil
}

// --- list_payments ---

type listPaymentsTool struct {
	sessions SessionStorer
	searcher CustomerSearcher
	payments PaymentRecorder
}

func (t *listPaymentsTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "list_payments",
		Desc: "列出指定客户的所有收款记录。客户名需精确匹配。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"customer_id":   {Type: schema.String, Desc: "客户ID（优先于customer_name）"},
			"customer_name": {Type: schema.String, Desc: "客户名称（精确匹配）"},
		}),
	}, nil
}

func (t *listPaymentsTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		CustomerID   string `json:"customer_id"`
		CustomerName string `json:"customer_name"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", NewToolError(CodeInvalidArgs, fmt.Sprintf("list_payments: parse args: %v", err))
	}
	id := strings.TrimSpace(p.CustomerID)
	if id == "" {
		name := strings.TrimSpace(p.CustomerName)
		if name == "" {
			return "", NewToolError(CodeInvalidArgs, "customer_id or customer_name is required")
		}
		results, err := t.searcher.Search(ctx, name, 3)
		if err != nil {
			return "", fmt.Errorf("list_payments: search: %w", err)
		}
		if len(results) == 0 || results[0].Score > 0 {
			return "", NewToolError(CodeNotFound,
				fmt.Sprintf("找不到精确匹配的客户 %q", name)).
				WithContext(map[string]any{"candidates": results})
		}
		id = results[0].ID
	}
	payments, err := t.payments.ListByCustomer(ctx, id)
	if err != nil {
		return "", fmt.Errorf("list_payments: %w", err)
	}
	total, _ := t.payments.TotalByCustomer(ctx, id)
	return mustJSON(map[string]any{
		"customer_id": id,
		"payments":    payments,
		"count":       len(payments),
		"total_paid":  total,
	}), nil
}

// --- add_customer_alias ---

type addCustomerAliasTool struct {
	searcher CustomerSearcher
}

func (t *addCustomerAliasTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "add_customer_alias",
		Desc: "为已有客户添加一个别名/简称。日后用该别名也能精确匹配到客户。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"customer_id": {Type: schema.String, Desc: "客户ID", Required: true},
			"alias":       {Type: schema.String, Desc: "别名", Required: true},
		}),
	}, nil
}

func (t *addCustomerAliasTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		CustomerID string `json:"customer_id"`
		Alias      string `json:"alias"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", NewToolError(CodeInvalidArgs, fmt.Sprintf("add_customer_alias: parse args: %v", err))
	}
	if err := t.searcher.AddAlias(ctx, CustomerAliasInput{CustomerID: p.CustomerID, Alias: p.Alias}); err != nil {
		if te := FromDomainError(err); te != nil {
			return "", te
		}
		return "", fmt.Errorf("add_customer_alias: %w", err)
	}
	return mustJSON(map[string]string{"message": "别名已添加", "customer_id": p.CustomerID, "alias": p.Alias}), nil
}

// --- add_product_alias ---

type addProductAliasTool struct {
	products ProductSearcher
}

func (t *addProductAliasTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "add_product_alias",
		Desc: "为已有商品添加一个别名/简称。日后用该别名也能精确匹配到商品。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"product_id": {Type: schema.String, Desc: "商品ID", Required: true},
			"alias":      {Type: schema.String, Desc: "别名", Required: true},
		}),
	}, nil
}

func (t *addProductAliasTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var p struct {
		ProductID string `json:"product_id"`
		Alias     string `json:"alias"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", NewToolError(CodeInvalidArgs, fmt.Sprintf("add_product_alias: parse args: %v", err))
	}
	if err := t.products.AddAlias(ctx, ProductAliasInput{ProductID: p.ProductID, Alias: p.Alias}); err != nil {
		if te := FromDomainError(err); te != nil {
			return "", te
		}
		return "", fmt.Errorf("add_product_alias: %w", err)
	}
	return mustJSON(map[string]string{"message": "别名已添加", "product_id": p.ProductID, "alias": p.Alias}), nil
}
