package handler

import (
	"net/http"

	"malta/internal/transport/http/shared"
)

// Invalid handlers.
type Invalid struct {
	Writer shared.Writer
}

// NotFound handler.
func (i *Invalid) NotFound(w http.ResponseWriter, r *http.Request) {
	i.Writer.Error(w, "endpoint not found", nil, http.StatusNotFound)
}

// MethodNotAllowed handler.
func (i *Invalid) MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	i.Writer.Error(w, "method not allowed", nil, http.StatusBadRequest)
}
