package vnet

import (
	"net"
	"strconv"
)

// ParseHost parses the host from a listen address like ":7079" or
// "127.0.0.1:7079".
func ParseHost(addr string) (string, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	return host, nil
}

// ParsePort parses the port number from a listen address like ":7079" or
// "127.0.0.1:7079".
func ParsePort(addr string) (int, error) {
	_, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(portText)
}

// MustParseHost returns the parsed listen host and panics when addr is
// invalid. It is intended for startup-time configuration paths that are
// expected to have already been validated.
func MustParseHost(addr string) string {
	host, err := ParseHost(addr)
	if err != nil {
		panic(err)
	}
	return host
}

// MustParsePort returns the parsed listen port and panics when addr is
// invalid. It is intended for startup-time configuration paths that are
// expected to have already been validated.
func MustParsePort(addr string) int {
	port, err := ParsePort(addr)
	if err != nil {
		panic(err)
	}
	return port
}
