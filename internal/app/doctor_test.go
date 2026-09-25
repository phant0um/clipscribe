package app

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/phant0um/clipscribe/internal/model"
)

// sha256("model bytes"), computed with shasum.
const modelBytesSHA = "9cb7487000bc86ac36ce83c4acfabe8878552be99572a6770f65ab1d048a5c48"

func TestDoctorAllGood(t *testing.T) { // AC7.1
	e := newEnv(t, "youtube-pt-nosubs.json")
	if code := e.run("doctor"); code != ExitOK {
		t.Fatalf("exit %d: %s", code, e.err)
	}
	for _, want := range []string{"yt-dlp", "ffmpeg", "whisper-cli", "model"} {
		if !strings.Contains(e.out.String(), want) {
			t.Errorf("report missing %s:\n%s", want, e.out)
		}
	}
}

func TestDoctorReportsFixes(t *testing.T) { // AC7.1
	e := newEnv(t, "youtube-pt-nosubs.json")
	e.d.LookPath = func(name string) (string, error) {
		if name == "whisper-cli" {
			return "", errors.New("not found")
		}
		return "/bin/" + name, nil
	}
	os.WriteFile(e.d.ConfigPath, []byte(`{"out_dir":"`+e.outDir+`","model_path":"`+e.tmpDir+`/none.bin"}`), 0o600)
	if code := e.run("doctor"); code != ExitMissing {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{"brew install whisper-cpp", "clipscribe doctor --install-model"} {
		if !strings.Contains(e.out.String(), want) {
			t.Errorf("report missing %q:\n%s", want, e.out)
		}
	}
}

func modelServer(t *testing.T, body string) *httptest.Server {
	s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
	t.Cleanup(s.Close)
	return s
}

func TestDoctorInstallModel(t *testing.T) { // AC7.3
	for _, c := range []struct {
		body string
		code int
	}{{"model bytes", ExitOK}, {"tampered!!!", ExitRuntime}} {
		e := newEnv(t, "youtube-pt-nosubs.json")
		dest := e.tmpDir + "/models/model.bin"
		os.WriteFile(e.d.ConfigPath, []byte(`{"out_dir":"`+e.outDir+`","model_path":"`+dest+`"}`), 0o600)
		s := modelServer(t, c.body)
		e.d.HTTPClient = s.Client()
		e.d.Model = model.Spec{URL: s.URL, SHA256: modelBytesSHA, Size: 11}
		if code := e.run("doctor", "--install-model"); code != c.code {
			t.Fatalf("%q: exit %d: %s", c.body, code, e.err)
		}
		_, err := os.Stat(dest)
		if (c.code == ExitOK) != (err == nil) {
			t.Errorf("%q: model present = %v", c.body, err == nil)
		}
	}
}
