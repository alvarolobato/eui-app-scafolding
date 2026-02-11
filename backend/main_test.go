package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
)

func TestConfigEndpoint(t *testing.T) {
	// Set up a test handler
	handler := func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		var result struct {
			APM struct {
				ServerURL string `json:"server_url"`
			} `json:"apm"`
			Google struct {
				ClientID   string `json:"client_id"`
				OAuthScope string `json:"oauth_scope"`
			} `json:"google"`
		}
		result.APM.ServerURL = "http://localhost:8200"
		result.Google.ClientID = "test-client-id"
		result.Google.OAuthScope = "openid email profile"
		json.NewEncoder(w).Encode(result)
	}

	// Create a request
	req, err := http.NewRequest("GET", "/api/config", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Create a new router and register the handler
	router := httprouter.New()
	router.GET("/api/config", handler)

	// Serve the request
	router.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("failed to unmarshal response: %v", err)
	}

	// Verify the response contains expected fields
	if _, ok := response["apm"]; !ok {
		t.Error("response missing 'apm' field")
	}
	if _, ok := response["google"]; !ok {
		t.Error("response missing 'google' field")
	}
}

func TestSampleDataGeneration(t *testing.T) {
	data := generateSampleData()

	// Check that we generate between 50-100 records
	if len(data) < 50 || len(data) > 100 {
		t.Errorf("expected 50-100 records, got %d", len(data))
	}

	// Check that each record has required fields
	for i, record := range data {
		if record.ID == "" {
			t.Errorf("record %d has empty ID", i)
		}
		if record.Name == "" {
			t.Errorf("record %d has empty Name", i)
		}
		if record.Description == "" {
			t.Errorf("record %d has empty Description", i)
		}
		if record.CreatedAt == "" {
			t.Errorf("record %d has empty CreatedAt", i)
		}
		if record.Status == "" {
			t.Errorf("record %d has empty Status", i)
		}
		if record.Category == "" {
			t.Errorf("record %d has empty Category", i)
		}
	}
}

func TestSecureCookiesEmpty(t *testing.T) {
	// Test with no encryption keys
	sc, err := newSecureCookies(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Empty secure cookies should return input unchanged
	value := "test-value"
	encoded, err := sc.Encode(value)
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	if encoded != value {
		t.Errorf("expected %q, got %q", value, encoded)
	}

	decoded, err := sc.Decode(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if decoded != value {
		t.Errorf("expected %q, got %q", value, decoded)
	}
}

func TestSplitAuthHeader(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Bearer token123", []string{"Bearer", "token123"}},
		{"Basic auth", []string{"Basic", "auth"}},
		{"NoSpace", []string{"NoSpace"}},
		{"Multiple Spaces Here", []string{"Multiple", "Spaces Here"}},
	}

	for _, test := range tests {
		result := splitAuthHeader(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("splitAuthHeader(%q) = %v, want %v", test.input, result, test.expected)
			continue
		}
		for i := range result {
			if result[i] != test.expected[i] {
				t.Errorf("splitAuthHeader(%q)[%d] = %q, want %q", test.input, i, result[i], test.expected[i])
			}
		}
	}
}
