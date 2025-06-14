package http_action

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dukex/operion/internal/domain"
)

// HTTPRequestAction performs an HTTP request
type HTTPRequestAction struct {
    ID      string
    Method  string
    URL     string
    Headers map[string]string
    Body    string
    Timeout time.Duration
}


func NewHTTPRequestAction(config map[string]interface{}) (*HTTPRequestAction, error) {
    // ... decoding logic from map[string]interface{} ...
    return &HTTPRequestAction{
        ID: "http-action-1",
        Method: http.MethodPost,
        URL: "https://api.example.com/data",
        Body: `{"key": "{{.TriggerData.value}}"}`, // Example of templating
        Timeout: 30 * time.Second,
    }, nil
}

func (a *HTTPRequestAction) GetID() string   { return a.ID }
func (a *HTTPRequestAction) GetType() string { return "http" }
func (a *HTTPRequestAction) GetConfig() map[string]interface{} { /* ... */ return nil}
func (a *HTTPRequestAction) Validate() error { /* ... */ return nil}

func (a *HTTPRequestAction) Execute(ctx context.Context, input domain.ExecutionContext) (domain.ExecutionContext, error) {
    log.Printf("Executing HTTPRequestAction '%s' to URL '%s'", a.ID, a.URL)
    
    // 1. Template the URL, Headers, and Body with data from the ExecutionContext
    //    (e.g., using Go's text/template package)
    //
    //    templatedURL, err := template.New("url").Parse(a.URL).Execute(...)
    
    // 2. Create an http.Request with a context that respects the timeout
    reqCtx, cancel := context.WithTimeout(ctx, a.Timeout)
    defer cancel()

    req, err := http.NewRequestWithContext(reqCtx, a.Method, a.URL, nil /* templated body */)
    if err != nil {
        return input, fmt.Errorf("failed to create http request: %w", err)
    }
    // Add headers...

    // 3. Execute the request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return input, fmt.Errorf("http request failed: %w", err)
    }
    defer resp.Body.Close()

    // 4. Process the response
    // ... read resp.Body ...

    // 5. Add results to the ExecutionContext
    if input.StepResults == nil {
        input.StepResults = make(map[string]interface{})
    }
    input.StepResults[a.ID] = map[string]interface{}{
        "status_code": resp.StatusCode,
        "body":        "response_body_content", // The actual content
    }
    
    log.Printf("HTTPRequestAction '%s' completed with status %d", a.ID, resp.StatusCode)
    return input, nil
}