package handler

import "github.com/teemuteemu/caddy-language-server/internal/document"

// Handler holds references to shared server state.
type Handler struct {
	store *document.Store
}

// New creates a Handler backed by the given document store.
func New(store *document.Store) *Handler {
	return &Handler{store: store}
}
