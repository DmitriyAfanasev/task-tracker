package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"secunda/internal/delivery/http/openapi"
	"secunda/internal/domain"
)

// writeError приводит любую ошибку к единому формату ErrorResponse.
func writeError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	domainErr, ok := isDomainError(err)
	if !ok {
		log.Error("unhandled error",
			"error", err,
			"request_id", requestIDFromContext(r.Context()),
		)
		domainErr = domain.ErrInternal.WithErr(err)
	}

	if domainErr.HTTPStatus >= http.StatusInternalServerError {
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

	writeJSON(w, domainErr.HTTPStatus, openapi.ErrorResponse{Error: body})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
