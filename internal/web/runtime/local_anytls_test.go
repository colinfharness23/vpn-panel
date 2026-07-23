package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestLocalAnyTLSMutationsUseSidecarReconcile(t *testing.T) {
	calls := 0
	local := NewLocal(LocalDeps{
		APIPort: func() int {
			t.Fatal("AnyTLS mutation unexpectedly requested the Xray API port")
			return 0
		},
		ReconcileSidecars: func() error {
			calls++
			return nil
		},
	})
	inbound := &model.Inbound{Protocol: model.AnyTLS}

	operations := []struct {
		name string
		run  func() error
	}{
		{"add inbound", func() error { return local.AddInbound(context.Background(), inbound) }},
		{"delete inbound", func() error { return local.DelInbound(context.Background(), inbound) }},
		{"update inbound", func() error { return local.UpdateInbound(context.Background(), inbound, inbound) }},
		{"add user", func() error {
			return local.AddUser(context.Background(), inbound, map[string]any{"email": "customer@example.com"})
		}},
		{"remove user", func() error { return local.RemoveUser(context.Background(), inbound, "customer@example.com") }},
		{"update user", func() error {
			return local.UpdateUser(context.Background(), inbound, "customer@example.com", model.Client{
				Email:    "customer@example.com",
				Password: "secret",
				Enable:   true,
			})
		}},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			before := calls
			if err := operation.run(); err != nil {
				t.Fatalf("%s: %v", operation.name, err)
			}
			if calls <= before {
				t.Fatalf("%s did not reconcile the AnyTLS sidecar", operation.name)
			}
		})
	}
}

func TestLocalAnyTLSPropagatesSidecarFailure(t *testing.T) {
	want := errors.New("sidecar unavailable")
	local := NewLocal(LocalDeps{ReconcileSidecars: func() error { return want }})
	err := local.AddUser(context.Background(), &model.Inbound{Protocol: model.AnyTLS}, nil)
	if !errors.Is(err, want) {
		t.Fatalf("AddUser error = %v, want %v", err, want)
	}
}
