package product

import (
	"context"
	"testing"

	"github.com/cccvno1/goplate/pkg/errkit"
)

func TestAddAlias_InvalidInput(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name string
		in   AddAliasInput
	}{
		{name: "missing product id", in: AddAliasInput{Alias: "红富士"}},
		{name: "missing alias", in: AddAliasInput{ProductID: "prod-1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.AddAlias(context.Background(), tt.in)
			if err == nil {
				t.Fatal("expected error")
			}
			if e := errkit.AsError(err); e == nil || e.Code != errkit.InvalidInput {
				t.Fatalf("expected invalid input error, got %v", err)
			}
		})
	}
}
