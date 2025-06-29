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
		`<p>`, `</p>`,
		`<a>`, `</a>`,
		`<i>`, `</i>`,
		`<em>`, `</em>`,
		`<strong>`, `</strong>`,
	}
	for _, remove := range removals {
		in = strings.ReplaceAll(in, remove, "")
	}
	replaceWithSpaces := []string{`<br>`, `<br/>`, `<br />`}
	for _, replace := range replaceWithSpaces {
		in = strings.ReplaceAll(in, replace, " ")
	}
	// Misc fixes, html/template takes care of escaping
	in = strings.ReplaceAll(in, "&amp;", "&")
	return strings.TrimSpace(in)
}
