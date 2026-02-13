package prompt

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnhanceViaAPI_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/enhance" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var req struct {
			Intent  string `json:"intent"`
			Format  string `json:"format"`
			Persona string `json:"persona"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.Intent != "review my code" || req.Format != "xml" {
			t.Errorf("unexpected body: intent=%q format=%q", req.Intent, req.Format)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"prompt": "<context>Expert reviewer</context>\n<task>Review the code</task>",
		})
	}))
	defer server.Close()

	result, err := EnhanceViaAPI(server.URL, EnhanceConfig{
		Intent:       "review my code",
		OutputFormat: "xml",
	})
	if err != nil {
		t.Fatalf("EnhanceViaAPI err = %v", err)
	}
	if result.Prompt != "<context>Expert reviewer</context>\n<task>Review the code</task>" {
		t.Errorf("unexpected prompt: %q", result.Prompt)
	}
}

func TestEnhanceViaAPI_EmptyPromptInResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"prompt": ""})
	}))
	defer server.Close()

	_, err := EnhanceViaAPI(server.URL, EnhanceConfig{Intent: "x", OutputFormat: "xml"})
	if err == nil {
		t.Fatal("expected error for empty prompt in response")
	}
}

func TestEnhanceViaAPI_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	_, err := EnhanceViaAPI(server.URL, EnhanceConfig{Intent: "x", OutputFormat: "xml"})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestEnhanceViaAPI_413(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		w.Write([]byte("body too large"))
	}))
	defer server.Close()

	_, err := EnhanceViaAPI(server.URL, EnhanceConfig{Intent: "x", OutputFormat: "xml"})
	if err == nil {
		t.Fatal("expected error for 413 response")
	}
}

func TestEnhanceViaAPI_429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	_, err := EnhanceViaAPI(server.URL, EnhanceConfig{Intent: "x", OutputFormat: "xml"})
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
}

func TestEnhanceViaAPI_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	_, err := EnhanceViaAPI(server.URL, EnhanceConfig{Intent: "x", OutputFormat: "xml"})
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestEnhanceViaAPI_DefaultFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Format string `json:"format"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Format != "xml" {
			t.Errorf("default format should be xml, got %q", req.Format)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"prompt": "ok"})
	}))
	defer server.Close()

	_, err := EnhanceViaAPI(server.URL, EnhanceConfig{Intent: "x"})
	if err != nil {
		t.Fatal(err)
	}
}
