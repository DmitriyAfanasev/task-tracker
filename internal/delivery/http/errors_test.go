package httpapi

import (
	"net/http"
	"testing"

	"secunda/internal/domain"
)

func TestStatusForKind(t *testing.T) {
	tests := []struct {
		kind domain.Kind
		want int
	}{
		{domain.KindBadRequest, http.StatusBadRequest},
		{domain.KindPayloadTooLarge, http.StatusRequestEntityTooLarge},
		{domain.KindUnauthorized, http.StatusUnauthorized},
		{domain.KindForbidden, http.StatusForbidden},
		{domain.KindNotFound, http.StatusNotFound},
		{domain.KindConflict, http.StatusConflict},
		{domain.KindInternal, http.StatusInternalServerError},
		{domain.Kind(0), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		if got := statusForKind(tt.kind); got != tt.want {
			t.Errorf("statusForKind(%v) = %d, want %d", tt.kind, got, tt.want)
		}
	}
}
