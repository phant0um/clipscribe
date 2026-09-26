package media

import (
	"net/url"
	"regexp"
	"testing"
)

var fuzzID = regexp.MustCompile(`^([A-Za-z0-9_-]{11}|[0-9]{1,20})$`)

// FuzzParseURL checks the allowlist invariants for any input: an accepted
// URL is https, on an allowed host, has a well-formed ID, and parses again
// to the same target.
func FuzzParseURL(f *testing.F) {
	for _, s := range []string{
		"https://www.youtube.com/watch?v=qD0_yWgifDM", "https://youtu.be/qD0_yWgifDM",
		"https://x.com/poteto/status/2102050467505430555/video/1", "https://twitter.com/a/status/1",
		"https://youtube.com.evil.io/watch?v=qD0_yWgifDM", "http://youtu.be/qD0_yWgifDM",
		"https://u:p@x.com/a/status/1", "https://x.com:444/a/status/1", "--exec=id", "",
	} {
		f.Add(s)
	}
	allowed := map[string]bool{"www.youtube.com": true, "x.com": true}
	f.Fuzz(func(t *testing.T, raw string) {
		tg, err := ParseURL(raw)
		if err != nil {
			return
		}
		u, err := url.Parse(tg.URL)
		if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" || !allowed[u.Host] {
			t.Fatalf("%q -> unsafe URL %q", raw, tg.URL)
		}
		if !fuzzID.MatchString(tg.ID) {
			t.Fatalf("%q -> bad ID %q", raw, tg.ID)
		}
		again, err := ParseURL(tg.URL)
		if err != nil || again != tg {
			t.Fatalf("%q -> %+v, reparse %+v %v", raw, tg, again, err)
		}
	})
}
