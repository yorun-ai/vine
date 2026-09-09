package http

import (
	"fmt"
	"net/http"
	"strings"

	rpchttp "go.yorun.ai/vrpc/transport/http"
)

func encodeDestinationToHeader(header http.Header, destination string) {
	value := rpchttp.EncodeFields("destination", destination)
	if options := header.Get(HeaderRpcOptions); options != "" {
		value = options + "," + value
	}
	header.Set(HeaderRpcOptions, value)
}

// ConsumeDestinationFromHeader removes Vine's App-to-Link routing option.
// Other options are forwarded unchanged.
func ConsumeDestinationFromHeader(header http.Header) (string, error) {
	values := header.Values(HeaderRpcOptions)
	if len(values) == 0 {
		return "", nil
	}
	if len(values) != 1 {
		return "", fmt.Errorf("invalid request header %s", HeaderRpcOptions)
	}
	fields, err := rpchttp.DecodeFields(values[0])
	if err != nil {
		return "", fmt.Errorf("invalid request header %s", HeaderRpcOptions)
	}
	destination := fields["destination"]
	if destination == "" {
		return "", nil
	}
	remaining := make([]string, 0, len(fields)-1)
	for part := range strings.SplitSeq(values[0], ",") {
		name, _, _ := strings.Cut(part, "=")
		if strings.TrimSpace(name) != "destination" {
			remaining = append(remaining, part)
		}
	}
	header.Del(HeaderRpcOptions)
	if len(remaining) > 0 {
		header.Set(HeaderRpcOptions, strings.Join(remaining, ","))
	}
	return destination, nil
}
