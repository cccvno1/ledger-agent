package customer

import "testing"

func TestNewCustomerResponse_NilAliasesBecomesEmpty(t *testing.T) {
	c := &Customer{ID: "1", Name: "张三", Aliases: nil}
	resp := NewCustomerResponse(c)
	if resp.Aliases == nil {
		t.Error("Aliases should not be nil in response")
	}
	if len(resp.Aliases) != 0 {
		t.Errorf("Aliases = %v, want empty slice", resp.Aliases)
	}
}

func TestNewCustomerResponse_Fields(t *testing.T) {
	c := &Customer{ID: "abc", Name: "李四", Aliases: []string{"小李", "李老板"}}
	resp := NewCustomerResponse(c)
	if resp.ID != "abc" {
		t.Errorf("ID = %q, want %q", resp.ID, "abc")
	}
	if resp.Name != "李四" {
		t.Errorf("Name = %q, want %q", resp.Name, "李四")
	}
	if len(resp.Aliases) != 2 {
		t.Errorf("len(Aliases) = %d, want 2", len(resp.Aliases))
	}
}
