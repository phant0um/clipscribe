package media

import "testing"

func TestParseURLAccepts(t *testing.T) {
	cases := []struct {
		in       string
		platform Platform
		id       string
	}{
		{"https://www.youtube.com/watch?v=qD0_yWgifDM", YouTube, "qD0_yWgifDM"},
		{"https://youtube.com/watch?v=qD0_yWgifDM&t=30", YouTube, "qD0_yWgifDM"},
		{"https://m.youtube.com/watch?v=qD0_yWgifDM", YouTube, "qD0_yWgifDM"},
		{"https://youtu.be/qD0_yWgifDM", YouTube, "qD0_yWgifDM"},
		{"https://www.youtube.com/shorts/qD0_yWgifDM", YouTube, "qD0_yWgifDM"},
		{"https://x.com/poteto/status/2102050467505430555", X, "2102050467505430555"},
		{"https://x.com/poteto/status/2102050467505430555/video/1", X, "2102050467505430555"},
		{"https://twitter.com/poteto/status/2102050467505430555", X, "2102050467505430555"},
		{"https://mobile.twitter.com/poteto/status/2102050467505430555", X, "2102050467505430555"},
	}
	for _, c := range cases {
		got, err := ParseURL(c.in)
		if err != nil {
			t.Errorf("ParseURL(%q) error: %v", c.in, err)
			continue
		}
		if got.Platform != c.platform || got.ID != c.id {
			t.Errorf("ParseURL(%q) = %+v, want %s %s", c.in, got, c.platform, c.id)
		}
	}
}

func TestParseURLRejects(t *testing.T) {
	for _, in := range []string{
		"",
		"http://www.youtube.com/watch?v=qD0_yWgifDM",      // not https
		"https://youtube.com.evil.io/watch?v=qD0_yWgifDM", // lookalike host
		"https://evilyoutube.com/watch?v=qD0_yWgifDM",     // suffix trick
		"https://user:pw@youtube.com/watch?v=qD0_yWgifDM", // userinfo
		"https://www.youtube.com/watch",                   // no id
		"https://www.youtube.com/watch?v=--exec=rm",       // flag-like id
		"https://www.youtube.com/playlist?list=PL123",     // playlist
		"https://x.com/poteto",                            // profile, no status
		"https://x.com/poteto/status/abc",                 // non-numeric id
		"https://vimeo.com/123",                           // other site
		"file:///etc/passwd",
		"--exec=touch /tmp/pwned",
	} {
		if got, err := ParseURL(in); err == nil {
			t.Errorf("ParseURL(%q) = %+v, want error", in, got)
		}
	}
}
