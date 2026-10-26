package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"secunda/internal/delivery/http/openapi"
	"secunda/internal/domain"
)

// writeError приводит любую ошибку к единому формату ErrorResponse.
// HTTP-статус выбирается здесь, а не в домене.
func writeError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	domainErr, ok := isDomainError(err)
	if !ok {
		log.Error("unhandled error",
			"error", err,
			"request_id", requestIDFromContext(r.Context()),
		)
		domainErr = domain.ErrInternal.WithErr(err)
	}

	status := statusForKind(domainErr.Kind)
	if status >= http.StatusInternalServerError {
		log.Error("request failed",
			"error", domainErr,
			"code", domainErr.Code,
			"request_id", requestIDFromContext(r.Context()),
		)
	}

	body := openapi.ErrorBody{
		Code:      domainErr.Code,
		Message:   domainErr.Message,
		RequestId: requestIDFromContext(r.Context()),
	}
	if len(domainErr.Details) > 0 {
		details := domainErr.Details
		body.Details = &details
	}

	writeJSON(w, status, openapi.ErrorResponse{Error: body})
}

func statusForKind(kind domain.Kind) int {
	switch kind {
	case domain.KindBadRequest:
		return http.StatusBadRequest
	case domain.KindPayloadTooLarge:
		return http.StatusRequestEntityTooLarge
	case domain.KindUnauthorized:
		return http.StatusUnauthorized
	case domain.KindForbidden:
		return http.StatusForbidden
	case domain.KindNotFound:
		return http.StatusNotFound
	case domain.KindConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func isMaxBytesError(err error) bool {
	var maxBytesErr *http.MaxBytesError
	return errors.As(err, &maxBytesErr)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
