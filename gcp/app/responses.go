package app

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

// RenderJSON renders response in JSON format
func RenderJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Error("failed to output JSON", map[string]interface{}{
			log.FnError: err.Error(),
		})
	}
}

// RenderError renders response as error
func RenderError(ctx context.Context, w http.ResponseWriter, e APIError) {
	fields := well.FieldsFromContext(ctx)
	fields["status"] = e.Status
	fields[log.FnError] = e.Error()
	log.Error(http.StatusText(e.Status), fields)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Status)
	err := json.NewEncoder(w).Encode(fields)
	if err != nil {
		log.Error("failed to output JSON", map[string]interface{}{
			log.FnError: err.Error(),
		})
	}
}
