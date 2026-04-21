package boot

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/cccvno1/goplate/pkg/appkit"
	"github.com/cccvno1/ledger-agent/internal/base/conf"
	"github.com/cccvno1/ledger-agent/internal/chat"
	"github.com/cccvno1/ledger-agent/internal/customer"
	"github.com/cccvno1/ledger-agent/internal/ledger"
	"github.com/cccvno1/ledger-agent/internal/payment"
	"github.com/cccvno1/ledger-agent/internal/product"
	"github.com/cccvno1/ledger-agent/internal/wechat"
)

// Run initializes all components and runs the application until shutdown.
func Run(ctx context.Context, logger *slog.Logger, cfg *conf.Config) error {
	db, err := openDB(cfg)
	if err != nil {
		return fmt.Errorf("boot: open db: %w", err)
	}

	app := appkit.New(logger)
	mux := http.NewServeMux()

	customerSvc := customer.Wire(mux, db)
	ledgerSvc := ledger.Wire(mux, db)
	productSvc := product.Wire(mux, db)
	paymentSvc := payment.Wire(mux, db)

	chatSvc, err := chat.Wire(ctx, mux, db, cfg.MiniMax,
		&customerAdapter{svc: customerSvc},
		&ledgerAdapter{svc: ledgerSvc},
		&ledgerAdapter{svc: ledgerSvc},
		&productAdapter{svc: productSvc},
		&paymentAdapter{svc: paymentSvc},
	)
	if err != nil {
		return fmt.Errorf("boot: chat wire: %w", err)
	}

	if err := wechat.Wire(ctx, mux, logger, &chatServiceAdapter{svc: chatSvc}); err != nil {
		return fmt.Errorf("boot: wechat wire: %w", err)
	}

	if err := wireHTTP(ctx, app, logger, cfg, mux); err != nil {
		return err
	}
	return app.Run(ctx)
}

func openDB(cfg *conf.Config) (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}
	return db, nil
}

// customerAdapter adapts customer.Service to chat.CustomerSearcher.
type customerAdapter struct {
	svc *customer.Service
}

// chatServiceAdapter adapts chat.Service to wechat.Chatter.
type chatServiceAdapter struct {
	svc *chat.Service
}

func (a *chatServiceAdapter) Chat(ctx context.Context, sessionID string, message string) (string, error) {
	out, err := a.svc.Chat(ctx, chat.ChatInput{SessionID: sessionID, Message: message})
	if err != nil {
		return "", err
	}
	return out.Reply, nil
}

func (a *customerAdapter) Search(ctx context.Context, query string, topN int) ([]chat.CustomerMatch, error) {
	results, err := a.svc.Search(ctx, customer.SearchInput{Query: query, TopN: topN})
	if err != nil {
		return nil, err
	}
	out := make([]chat.CustomerMatch, len(results))
	for i, r := range results {
		out[i] = chat.CustomerMatch{ID: r.Customer.ID, Name: r.Customer.Name, Score: r.Score}
	}
	return out, nil
}

func (a *customerAdapter) ListAll(ctx context.Context) ([]chat.CustomerRef, error) {
	customers, err := a.svc.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]chat.CustomerRef, len(customers))
	for i, c := range customers {
		out[i] = chat.CustomerRef{ID: c.ID, Name: c.Name}
	}
	return out, nil
}

func (a *customerAdapter) Create(ctx context.Context, name string) (chat.CustomerRef, error) {
	c, err := a.svc.Create(ctx, customer.CreateInput{Name: name})
	if err != nil {
		return chat.CustomerRef{}, err
	}
	return chat.CustomerRef{ID: c.ID, Name: c.Name}, nil
}

func (a *customerAdapter) AddAlias(ctx context.Context, in chat.CustomerAliasInput) error {
	return a.svc.AddAlias(ctx, customer.AddAliasInput{
		CustomerID: in.CustomerID,
		Alias:      in.Alias,
	})
}

// ledgerAdapter adapts ledger.Service to chat.LedgerWriter and chat.LedgerQuerier.
type ledgerAdapter struct {
	svc *ledger.Service
}

func (a *ledgerAdapter) Create(ctx context.Context, in chat.LedgerCreateInput) (chat.LedgerEntryRef, error) {
	e, err := a.svc.Create(ctx, ledger.CreateInput{
		CustomerID:   in.CustomerID,
		CustomerName: in.CustomerName,
		ProductName:  in.ProductName,
		UnitPrice:    in.UnitPrice,
		Quantity:     in.Quantity,
		Unit:         in.Unit,
		EntryDate:    in.EntryDate,
		Notes:        in.Notes,
	})
	if err != nil {
		return chat.LedgerEntryRef{}, err
	}
	return entryToRef(e), nil
}

func (a *ledgerAdapter) Update(ctx context.Context, in chat.LedgerUpdateInput) (chat.LedgerEntryRef, error) {
	e, err := a.svc.Update(ctx, ledger.UpdateInput{
		ID:          in.ID,
		ProductName: in.ProductName,
		UnitPrice:   in.UnitPrice,
		Quantity:    in.Quantity,
		Unit:        in.Unit,
		EntryDate:   in.EntryDate,
		Notes:       in.Notes,
	})
	if err != nil {
		return chat.LedgerEntryRef{}, err
	}
	return entryToRef(e), nil
}

func (a *ledgerAdapter) Delete(ctx context.Context, entryID string) error {
	return a.svc.Delete(ctx, ledger.DeleteInput{ID: entryID})
}

func (a *ledgerAdapter) SettleByCustomer(ctx context.Context, customerID string) error {
	return a.svc.SettleByCustomer(ctx, customerID)
}

func (a *ledgerAdapter) List(ctx context.Context, in chat.LedgerListInput) ([]chat.LedgerEntryRef, error) {
	entries, err := a.svc.List(ctx, ledger.ListInput{
		CustomerID:   in.CustomerID,
		CustomerName: in.CustomerName,
		DateFrom:     in.DateFrom,
		DateTo:       in.DateTo,
		IsSettled:    in.IsSettled,
	})
	if err != nil {
		return nil, err
	}
	out := make([]chat.LedgerEntryRef, len(entries))
	for i, e := range entries {
		out[i] = entryToRef(e)
	}
	return out, nil
}

func (a *ledgerAdapter) SummaryByCustomer(ctx context.Context, customerID string) ([]chat.LedgerSummaryRef, error) {
	summaries, err := a.svc.SummaryByCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	out := make([]chat.LedgerSummaryRef, len(summaries))
	for i, s := range summaries {
		out[i] = chat.LedgerSummaryRef{
			CustomerID:    s.CustomerID,
			CustomerName:  s.CustomerName,
			TotalAmount:   s.TotalAmount,
			PendingAmount: s.PendingAmount,
			EntryCount:    s.EntryCount,
		}
	}
	return out, nil
}

func entryToRef(e *ledger.Entry) chat.LedgerEntryRef {
	return chat.LedgerEntryRef{
		ID:           e.ID,
		CustomerName: e.CustomerName,
		ProductName:  e.ProductName,
		UnitPrice:    e.UnitPrice,
		Quantity:     e.Quantity,
		Amount:       e.Amount,
		Unit:         e.Unit,
		EntryDate:    e.EntryDate,
		IsSettled:    e.IsSettled,
		Notes:        e.Notes,
	}
}

// productAdapter adapts product.Service to chat.ProductSearcher.
type productAdapter struct {
	svc *product.Service
}

func (a *productAdapter) Search(ctx context.Context, query string, topN int) ([]chat.ProductMatch, error) {
	results, err := a.svc.Search(ctx, query, topN)
	if err != nil {
		return nil, err
	}
	out := make([]chat.ProductMatch, len(results))
	for i, r := range results {
		out[i] = chat.ProductMatch{ID: r.Product.ID, Name: r.Product.Name, Score: r.Score, MatchedVia: r.MatchedVia}
	}
	return out, nil
}

func (a *productAdapter) FindOrCreate(ctx context.Context, name string) (chat.ProductRef, error) {
	p, err := a.svc.FindOrCreate(ctx, name)
	if err != nil {
		return chat.ProductRef{}, err
	}
	return chat.ProductRef{ID: p.ID, Name: p.Name, DefaultUnit: p.DefaultUnit, ReferencePrice: p.ReferencePrice}, nil
}

func (a *productAdapter) AddAlias(ctx context.Context, in chat.ProductAliasInput) error {
	return a.svc.AddAlias(ctx, product.AddAliasInput{
		ProductID: in.ProductID,
		Alias:     in.Alias,
	})
}

// paymentAdapter adapts payment.Service to chat.PaymentRecorder.
type paymentAdapter struct {
	svc *payment.Service
}

func (a *paymentAdapter) Create(ctx context.Context, in chat.PaymentCreateInput) (chat.PaymentRef, error) {
	p, err := a.svc.Create(ctx, payment.CreateInput{
		CustomerID:  in.CustomerID,
		Amount:      in.Amount,
		PaymentDate: in.PaymentDate,
		Notes:       in.Notes,
	})
	if err != nil {
		return chat.PaymentRef{}, err
	}
	return chat.PaymentRef{ID: p.ID, Amount: p.Amount, PaymentDate: p.PaymentDate}, nil
}

func (a *paymentAdapter) TotalByCustomer(ctx context.Context, customerID string) (float64, error) {
	return a.svc.TotalByCustomer(ctx, customerID)
}
