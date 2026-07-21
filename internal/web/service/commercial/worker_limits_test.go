package commercial

import (
	"reflect"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

func TestProvisionInboundIDsRequiresExplicitPlanBinding(t *testing.T) {
	worker := NewWorker()

	ids, err := worker.provisionInboundIDs(&model.Plan{})
	if err != nil {
		t.Fatalf("empty binding returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("empty binding granted inbounds %v; want no access", ids)
	}

	ids, err = worker.provisionInboundIDs(&model.Plan{ProvisionInboundIDs: `[7,3,7]`})
	if err != nil {
		t.Fatalf("explicit binding returned error: %v", err)
	}
	if !reflect.DeepEqual(ids, []int{3, 7}) {
		t.Fatalf("explicit binding = %v, want [3 7]", ids)
	}

	if _, err := worker.provisionInboundIDs(&model.Plan{ProvisionInboundIDs: `not-json`}); err == nil {
		t.Fatal("malformed binding should be rejected")
	}
}
