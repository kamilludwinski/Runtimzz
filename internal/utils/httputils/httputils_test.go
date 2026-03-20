package httputils_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kamilludwinski/runtimzzz/internal/utils/httputils"
)

func TestDownloadFile_Success(t *testing.T) {
	expected := "hello world"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expected))
	}))
	defer server.Close()

	path, err := httputils.DownloadFile(nil, server.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed reading downloaded file: %v", err)
	}

	if string(data) != expected {
		t.Fatalf("unexpected file content: got %q want %q", data, expected)
	}
}

func TestDownloadFile_EmptyURL(t *testing.T) {
	_, err := httputils.DownloadFile(nil, "")
	if err == nil {
		t.Fatal("expected error for empty url")
	}
}

func TestDownloadFile_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	_, err := httputils.DownloadFile(nil, server.URL)
	if err == nil {
		t.Fatal("expected error for non 200 status")
	}
}

func TestDownloadFile_ClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close() // conn failure

	_, err := httputils.DownloadFile(nil, server.URL)
	if err == nil {
		t.Fatal("expected client error")
	}
}

func TestDownloadFile_CustomClient(t *testing.T) {
	expected := "custom client test"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(expected))
	}))
	defer server.Close()

	client := server.Client()

	path, err := httputils.DownloadFile(client, server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed reading file: %v", err)
	}

	if string(data) != expected {
		t.Fatalf("unexpected content: got %q want %q", data, expected)
	}
}
