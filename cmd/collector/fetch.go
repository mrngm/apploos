package main

import (
	"fmt"
	"strings"
)

var (
	SupportedProtocols = []string{
		"http://", "https://", "file://",
	}
)

// IsSupportedSource returns the protocol and nil error if the given src is supported, or an appropriate message in
// error otherwise.
func IsSupportedSource(src string) (string, error) {
	if !strings.Contains(src, "://") {
		return "", fmt.Errorf("no protocol found, missing :// in %q", src)
	}

	for _, proto := range SupportedProtocols {
		if strings.HasPrefix(src, proto) {
			return proto, nil
		}
	}

	return "", fmt.Errorf("unsupported source protocol for value %q", src)
}

func FetchSource(src string) ([]byte, error) {
	protocol, err := IsSupportedSource(src)
	if err != nil {
		return nil, err
	}

	switch protocol {
	case "http://", "https://":

		return nil, fmt.Errorf("unimplemented")
	case "file://":

		return nil, fmt.Errorf("unimplemented")
	}

	return nil, fmt.Errorf("protocol %q seemed supported, but implementation is missing", protocol)
}

// vim: cc=120:
