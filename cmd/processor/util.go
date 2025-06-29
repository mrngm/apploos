package main

import (
	"regexp"
	"strings"
)

var (
	HTMLFilterAnchorRegexp = regexp.MustCompile(`(?is) href="[^"]+"`)
)

func cleanupHTML(in string) string {
	in = HTMLFilterAnchorRegexp.ReplaceAllString(in, "")
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
	return strings.TrimSpace(in)
}
