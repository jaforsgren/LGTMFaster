package common

import "testing"

func TestHTMLToText(t *testing.T) {
	tests := []struct {
		name string
		rich string
		want string
	}{
		{"plain text is kept", "Fix the login", "Fix the login"},
		{"paragraphs and breaks become lines", "<div>First</div><div>Second<br/>Third</div>", "First\nSecond\nThird"},
		{"list items become dashes", "<ul><li>one</li><li class=\"x\">two</li></ul>", "- one\n- two"},
		{"entities are decoded", "a &lt; b &amp;&amp;&nbsp;c", "a < b && c"},
		{"blank lines are collapsed", "<p>a</p><p></p><p></p><p>b</p>", "a\n\nb"},
		{"empty stays empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HTMLToText(tt.rich); got != tt.want {
				t.Errorf("HTMLToText(%q) = %q, want %q", tt.rich, got, tt.want)
			}
		})
	}
}
