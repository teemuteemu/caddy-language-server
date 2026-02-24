package server

import (
	"caddy-ls/internal/document"
	"caddy-ls/internal/handler"

	"github.com/tliron/commonlog"
	_ "github.com/tliron/commonlog/simple"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspServer "github.com/tliron/glsp/server"
)

// Run wires up the LSP handler and starts the server on stdio.
func Run(logLevel string) error {
	configureLogging(logLevel)

	store := document.New()
	h := handler.New(store)

	lspHandler := protocol.Handler{
		Initialize:             h.Initialize,
		Initialized:            h.Initialized,
		Shutdown:               h.Shutdown,
		SetTrace:               h.SetTrace,
		TextDocumentDidOpen:    h.DidOpen,
		TextDocumentDidChange:  h.DidChange,
		TextDocumentDidSave:    h.DidSave,
		TextDocumentDidClose:   h.DidClose,
		TextDocumentCompletion: h.Completion,
		TextDocumentHover:      h.Hover,
	}

	s := glspServer.NewServer(&lspHandler, "caddy-ls", false)
	return s.RunStdio()
}

func configureLogging(level string) {
	var logLevel commonlog.Level
	switch level {
	case "debug":
		logLevel = commonlog.Debug
	case "info":
		logLevel = commonlog.Info
	case "warning", "warn":
		logLevel = commonlog.Warning
	case "error":
		logLevel = commonlog.Error
	default:
		logLevel = commonlog.Warning
	}

	commonlog.Configure(1, nil)
	_ = logLevel
}
