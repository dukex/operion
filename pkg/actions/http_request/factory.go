package http_request

import "github.com/dukex/operion/pkg/protocol"

func NewHTTPRequestActionFactory() *HTTPRequestActionFactory {
	return &HTTPRequestActionFactory{}
}

type HTTPRequestActionFactory struct{}

func (h *HTTPRequestActionFactory) Create(config map[string]interface{}) (protocol.Action, error) {
	return NewHTTPRequestAction(config)
}

func (h *HTTPRequestActionFactory) ID() string {
	return "http_request"
}
