package transcript

import (
	"reflect"
	"testing"
	"time"
)

func seg(from, to float64, text string) Segment {
	return Segment{Start: time.Duration(from * float64(time.Second)), End: time.Duration(to * float64(time.Second)), Text: text}
}

func para(at float64, text string) Paragraph {
	return Paragraph{Start: time.Duration(at * float64(time.Second)), Text: text}
}

func TestParagraphs(t *testing.T) {
	cases := []struct {
		name string
		in   []Segment
		want []Paragraph
	}{
		{"empty", nil, nil},
		{"pause over 1.5s breaks", []Segment{seg(0, 2, "a"), seg(4, 5, "b")},
			[]Paragraph{para(0, "a"), para(4, "b")}},
		{"pause of exactly 1.5s does not break", []Segment{seg(0, 2, "a"), seg(3.5, 5, "b")},
			[]Paragraph{para(0, "a b")}},
		{"45s cap breaks even without punctuation",
			[]Segment{seg(0, 10, "a"), seg(10, 20, "b"), seg(20, 30, "c"), seg(30, 40, "d"), seg(40, 50, "e"), seg(50, 55, "f")},
			[]Paragraph{para(0, "a b c d e"), para(50, "f")}},
		{"sentence end breaks only after 20s",
			[]Segment{seg(0, 10, "Um."), seg(10, 15, "dois"), seg(15, 21, "três?"), seg(21, 25, "quatro")},
			[]Paragraph{para(0, "Um. dois três?"), para(21, "quatro")}},
	}
	for _, c := range cases {
		if got := Paragraphs(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}
