package product

import "testing"

func TestNewProductResponse_NilAliasesBecomesEmpty(t *testing.T) {
	p := &Product{ID: "1", Name: "苹果", Aliases: nil, DefaultUnit: "斤"}
	resp := NewProductResponse(p)
	if resp.Aliases == nil {
		t.Error("Aliases should not be nil in response")
	}
	if len(resp.Aliases) != 0 {
		t.Errorf("Aliases = %v, want empty slice", resp.Aliases)
	}
}

func TestNewProductResponse_Fields(t *testing.T) {
	p := &Product{
		ID: "p1", Name: "苹果", Aliases: []string{"红富士"},
		DefaultUnit: "斤", ReferencePrice: 5.5,
	}
	resp := NewProductResponse(p)
	if resp.ID != "p1" {
		t.Errorf("ID = %q, want %q", resp.ID, "p1")
	}
	if resp.DefaultUnit != "斤" {
		t.Errorf("DefaultUnit = %q, want %q", resp.DefaultUnit, "斤")
	}
	if resp.ReferencePrice != 5.5 {
		t.Errorf("ReferencePrice = %v, want 5.5", resp.ReferencePrice)
	}
}
