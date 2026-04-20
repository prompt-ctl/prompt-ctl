package cloud

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNew_DisabledWhenNotOptedIn(t *testing.T) {
	c := New(false, "https://cloud.example.com")
	if c.Enabled() {
		t.Fatal("client should be disabled when opted out")
	}
	if err := c.PostRating(5, 120); err != nil {
		t.Fatalf("PostRating on disabled client should be no-op, got %v", err)
	}
	if err := c.SubmitFeedback("nice"); err != nil {
		t.Fatalf("SubmitFeedback on disabled client should be no-op, got %v", err)
	}
}

func TestHTTPClient_PostsRatingAndFeedback(t *testing.T) {
	var ratingCalls int32
	var feedbackCalls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rating":
			atomic.AddInt32(&ratingCalls, 1)
		case "/feedback":
			atomic.AddInt32(&feedbackCalls, 1)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(true, srv.URL)
	if !c.Enabled() {
		t.Fatal("client should be enabled")
	}
	if err := c.PostRating(4, 99); err != nil {
		t.Fatalf("PostRating err = %v", err)
	}
	if err := c.SubmitFeedback("good"); err != nil {
		t.Fatalf("SubmitFeedback err = %v", err)
	}
	if atomic.LoadInt32(&ratingCalls) != 1 {
		t.Fatalf("rating calls = %d, want 1", ratingCalls)
	}
	if atomic.LoadInt32(&feedbackCalls) != 1 {
		t.Fatalf("feedback calls = %d, want 1", feedbackCalls)
	}
}
