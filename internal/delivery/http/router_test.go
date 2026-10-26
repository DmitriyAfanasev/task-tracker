package httpapi

import (
	"testing"

	"secunda/internal/delivery/http/openapi"
)

func TestPublicOperationsAreDerivedFromSpec(t *testing.T) {
	spec, err := openapi.GetSpec()
	if err != nil {
		t.Fatalf("GetSpec() error = %v", err)
	}

	public := publicOperations(spec)

	wantPublic := []string{"POST /register", "POST /login"}
	for _, operation := range wantPublic {
		if _, ok := public[operation]; !ok {
			t.Errorf("operation %q must be public", operation)
		}
	}

	wantProtected := []string{"GET /teams", "POST /teams", "POST /teams/{id}/invite"}
	for _, operation := range wantProtected {
		if _, ok := public[operation]; ok {
			t.Errorf("operation %q must be protected", operation)
		}
	}
}
