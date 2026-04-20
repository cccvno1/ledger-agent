package main_test

import (
	"testing"

	"github.com/cccvno1/goplate/pkg/archkit"
)

func TestArchitecture(t *testing.T) {
	rules := []archkit.Rule{
		{
			Package: "domain",
			MustNot: []string{"base"},
		},
		{
			Package: "customer",
			MustNot: []string{"base"},
		},
		{
			Package: "ledger",
			MustNot: []string{"base"},
		},
		{
			Package: "product",
			MustNot: []string{"base"},
		},
		{
			Package: "payment",
			MustNot: []string{"base"},
		},
		{
			Package: "chat",
			MustNot: []string{"base", "customer", "ledger", "product", "payment"},
		},
		{
			Package: "wechat",
			MustNot: []string{"base", "customer", "ledger", "product", "payment", "chat"},
		},
	}

	violations, err := archkit.Check("./internal", "github.com/cccvno1/ledger-agent", rules)
	if err != nil {
		t.Fatalf("archkit check: %v", err)
	}

	for _, v := range violations {
		t.Errorf("architecture violation: %s imports %s (package %s must not import %s)",
			v.File, v.ImportPath, v.Package, v.Rule.Package)
	}
}
