package ridge_test

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/fujiwara/ridge"
)

func TestRESTAPIPayloadDetection(t *testing.T) {
	tests := []struct {
		name        string
		payload     string
		expectedType string
	}{
		{
			name: "REST API payload",
			payload: `{
				"resource": "/hello",
				"path": "/hello",
				"httpMethod": "GET",
				"headers": {
					"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"
				},
				"queryStringParameters": null,
				"pathParameters": null,
				"stageVariables": null,
				"requestContext": {
					"resourceId": "123456",
					"resourcePath": "/hello",
					"httpMethod": "GET",
					"requestId": "c6af9ac6-7b61-11e6-9a41-93e8deadbeef",
					"protocol": "HTTP/1.1",
					"stage": "prod",
					"apiId": "1234567890"
				},
				"body": null,
				"isBase64Encoded": false
			}`,
			expectedType: ridge.PayloadTypeRESTAPI,
		},
		{
			name: "HTTP API v1 payload",
			payload: `{
				"version": "1.0",
				"resource": "/hello",
				"path": "/hello",
				"httpMethod": "GET",
				"headers": {
					"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"
				},
				"queryStringParameters": null,
				"pathParameters": null,
				"stageVariables": null,
				"requestContext": {
					"resourceId": "123456",
					"resourcePath": "/hello",
					"httpMethod": "GET",
					"requestId": "c6af9ac6-7b61-11e6-9a41-93e8deadbeef",
					"protocol": "HTTP/1.1"
				},
				"body": null,
				"isBase64Encoded": false
			}`,
			expectedType: ridge.PayloadTypeHTTPAPIv1,
		},
		{
			name: "HTTP API v2 payload",
			payload: `{
				"version": "2.0",
				"routeKey": "GET /hello",
				"rawPath": "/hello",
				"rawQueryString": "",
				"cookies": [],
				"headers": {
					"Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"
				},
				"queryStringParameters": {},
				"requestContext": {
					"accountId": "123456789012",
					"apiId": "1234567890",
					"domainName": "1234567890.execute-api.us-east-1.amazonaws.com",
					"domainPrefix": "1234567890",
					"http": {
						"method": "GET",
						"path": "/hello",
						"protocol": "HTTP/1.1",
						"sourceIp": "203.0.113.1",
						"userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
					},
					"requestId": "c6af9ac6-7b61-11e6-9a41-93e8deadbeef",
					"routeKey": "GET /hello",
					"stage": "prod",
					"time": "09/Apr/2015:12:34:56 +0000",
					"timeEpoch": 1428582896000
				},
				"body": null,
				"pathParameters": null,
				"isBase64Encoded": false,
				"stageVariables": null
			}`,
			expectedType: ridge.PayloadTypeHTTPAPIv2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, apiType, err := ridge.NewRequestWithAPIType(json.RawMessage(tt.payload))
			if err != nil {
				t.Fatalf("failed to create request: %s", err)
			}
			if apiType != tt.expectedType {
				t.Errorf("expected API type %s, got %s", tt.expectedType, apiType)
			}
		})
	}
}

func TestRESTAPIResponseWriterNoCookies(t *testing.T) {
	w := ridge.NewResponseWriterWithAPIType(ridge.PayloadTypeRESTAPI)
	
	// Add some headers and cookies
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Set-Cookie", "session=abc123; HttpOnly")
	w.Header().Add("Set-Cookie", "user=john; Max-Age=3600")
	w.WriteHeader(200)
	
	_, err := io.WriteString(w, `{"message": "Hello World"}`)
	if err != nil {
		t.Error(err)
	}
	
	response := w.ResponseForAPIType()
	
	// Check that we get RestResponse type
	restResp, ok := response.(ridge.RestResponse)
	if !ok {
		t.Errorf("expected RestResponse type, got %T", response)
		return
	}
	
	// Verify response fields
	if restResp.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", restResp.StatusCode)
	}
	
	if restResp.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", restResp.Headers["Content-Type"])
	}
	
	if restResp.Body != `{"message": "Hello World"}` {
		t.Errorf("expected body %s, got %s", `{"message": "Hello World"}`, restResp.Body)
	}
	
	// Most importantly, verify that REST API response doesn't have cookies field
	// by checking that the JSON marshalling doesn't include cookies
	jsonBytes, err := json.Marshal(restResp)
	if err != nil {
		t.Error(err)
	}
	
	jsonString := string(jsonBytes)
	if contains(jsonString, "cookies") {
		t.Errorf("REST API response should not contain cookies field, but got: %s", jsonString)
	}
}

func TestHTTPAPIResponseWriterWithCookies(t *testing.T) {
	w := ridge.NewResponseWriterWithAPIType(ridge.PayloadTypeHTTPAPIv1)
	
	// Add some headers and cookies
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Set-Cookie", "session=abc123; HttpOnly")
	w.Header().Add("Set-Cookie", "user=john; Max-Age=3600")
	w.WriteHeader(200)
	
	_, err := io.WriteString(w, `{"message": "Hello World"}`)
	if err != nil {
		t.Error(err)
	}
	
	response := w.ResponseForAPIType()
	
	// Check that we get Response type
	httpResp, ok := response.(ridge.Response)
	if !ok {
		t.Errorf("expected Response type, got %T", response)
		return
	}
	
	// Verify response fields
	if httpResp.StatusCode != 200 {
		t.Errorf("expected status code 200, got %d", httpResp.StatusCode)
	}
	
	if httpResp.Headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", httpResp.Headers["Content-Type"])
	}
	
	if httpResp.Body != `{"message": "Hello World"}` {
		t.Errorf("expected body %s, got %s", `{"message": "Hello World"}`, httpResp.Body)
	}
	
	// Verify that HTTP API response has cookies
	if len(httpResp.Cookies) != 2 {
		t.Errorf("expected 2 cookies, got %d", len(httpResp.Cookies))
	}
	
	if httpResp.Cookies[0] != "session=abc123; HttpOnly" {
		t.Errorf("expected cookie session=abc123; HttpOnly, got %s", httpResp.Cookies[0])
	}
	
	if httpResp.Cookies[1] != "user=john; Max-Age=3600" {
		t.Errorf("expected cookie user=john; Max-Age=3600, got %s", httpResp.Cookies[1])
	}
	
	// Verify that HTTP API response JSON includes cookies field
	jsonBytes, err := json.Marshal(httpResp)
	if err != nil {
		t.Error(err)
	}
	
	jsonString := string(jsonBytes)
	if !contains(jsonString, "cookies") {
		t.Errorf("HTTP API response should contain cookies field, but got: %s", jsonString)
	}
}

func TestDefaultResponseWriter(t *testing.T) {
	// Test that default ResponseWriter (without API type) behaves like HTTP API
	w := ridge.NewResponseWriter()
	
	w.Header().Add("Set-Cookie", "test=value")
	w.WriteHeader(200)
	
	_, err := io.WriteString(w, "test")
	if err != nil {
		t.Error(err)
	}
	
	response := w.ResponseForAPIType()
	
	// Should default to HTTP API behavior (with cookies)
	httpResp, ok := response.(ridge.Response)
	if !ok {
		t.Errorf("expected Response type, got %T", response)
		return
	}
	
	if len(httpResp.Cookies) != 1 {
		t.Errorf("expected 1 cookie, got %d", len(httpResp.Cookies))
	}
}

func TestRESTAPIResponseWriterPanicsOnResponse(t *testing.T) {
	// Test that calling Response() on REST API ResponseWriter panics
	w := ridge.NewResponseWriterWithAPIType(ridge.PayloadTypeRESTAPI)
	
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	
	_, err := io.WriteString(w, `{"message": "Hello World"}`)
	if err != nil {
		t.Error(err)
	}
	
	// This should panic because REST API should not use Response() method
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when calling Response() on REST API ResponseWriter")
		} else {
			// Expected panic - test passes
			t.Logf("correctly panicked with: %v", r)
		}
	}()
	
	// This should panic
	_ = w.Response()
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAt(s, substr, 1))))
}

func containsAt(s, substr string, start int) bool {
	if start >= len(s) {
		return false
	}
	if start+len(substr) > len(s) {
		return containsAt(s, substr, start+1)
	}
	if s[start:start+len(substr)] == substr {
		return true
	}
	return containsAt(s, substr, start+1)
}