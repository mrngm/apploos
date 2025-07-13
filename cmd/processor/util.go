package main

import (
	"regexp"
	"strings"
)

var (
	HTMLFilterAnchorRegexp          = regexp.MustCompile(`(?is) href="[^"]+"`)
	HTMLFilterUnicodeEntitiesRegexp = regexp.MustCompile(`(?is)&#x[0-9a-f]+;`)
)

func cleanupHTML(in string) string {
	in = HTMLFilterAnchorRegexp.ReplaceAllString(in, "")
	in = HTMLFilterUnicodeEntitiesRegexp.ReplaceAllString(in, "")
	removals := []string{
		`<p>`,
		`<a>`, `</a>`,
		`<i>`, `</i>`,
		`<em>`, `</em>`,
		`<strong>`, `</strong>`,
	}
	for _, remove := range removals {
		in = strings.ReplaceAll(in, remove, "")
	}
	replaceWithNewlines := []string{`<br>`, `<br/>`, `<br />`, `</p>`}
	for _, replace := range replaceWithNewlines {
		in = strings.ReplaceAll(in, replace, "\n")
	}
	// Misc fixes, html/template takes care of escaping
	in = strings.ReplaceAll(in, "&amp;", "&")
	return strings.TrimSpace(in)
}
