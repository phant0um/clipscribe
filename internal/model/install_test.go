package model

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sha256("model bytes"), computed with shasum.
const modelBytesSHA = "9cb7487000bc86ac36ce83c4acfabe8878552be99572a6770f65ab1d048a5c48"

func serve(t *testing.T, body string) *httptest.Server {
	t.Helper()
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	t.Cleanup(s.Close)
	return s
}

func TestInstallVerifiesHashBeforeWriting(t *testing.T) {
	s := serve(t, "model bytes")
	dest := filepath.Join(t.TempDir(), "models", "m.bin")
	spec := Spec{URL: s.URL + "/m.bin", SHA256: modelBytesSHA, Size: 11}
	if err := Install(context.Background(), s.Client(), spec, dest); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(dest); string(b) != "model bytes" {
		t.Errorf("content %q", b)
	}
	if entries, _ := os.ReadDir(filepath.Dir(dest)); len(entries) != 1 {
		t.Errorf("leftovers %v", entries)
	}
}

func TestInstallRejectsWrongHash(t *testing.T) {
	s := serve(t, "tampered!!!")
	dest := filepath.Join(t.TempDir(), "m.bin")
	err := Install(context.Background(), s.Client(), Spec{URL: s.URL, SHA256: modelBytesSHA, Size: 11}, dest)
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("err = %v", err)
	}
	if entries, _ := os.ReadDir(filepath.Dir(dest)); len(entries) != 0 {
		t.Errorf("file kept after bad hash: %v", entries)
	}
}

func TestInstallRejectsWrongSizeAndPlainHTTP(t *testing.T) {
	s := serve(t, "model bytes plus more")
	dest := filepath.Join(t.TempDir(), "m.bin")
	if err := Install(context.Background(), s.Client(), Spec{URL: s.URL, SHA256: modelBytesSHA, Size: 11}, dest); err == nil {
		t.Error("oversized download accepted")
	}
	if err := Install(context.Background(), http.DefaultClient, Spec{URL: "http://example.com/m.bin", SHA256: modelBytesSHA, Size: 11}, dest); err == nil {
		t.Error("http URL accepted")
	}
}

func TestDefaultSpecMatchesResearch(t *testing.T) {
	if LargeV3Turbo.SHA256 != "1fc70f774d38eb169993ac391eea357ef47c88757ef72ee5943879b7e8e2bc69" || LargeV3Turbo.Size != 1624555275 {
		t.Errorf("spec = %+v", LargeV3Turbo)
	}
}
