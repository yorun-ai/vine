package webproxy

import (
	"errors"
	"net/http"
)

var errWebEndpointUnavailable = errors.New("web endpoint unavailable")

func writeInboundError(w http.ResponseWriter, err error) {
	if errors.Is(err, errWebEndpointUnavailable) {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	http.Error(w, err.Error(), http.StatusBadRequest)
}

func appendRequestQuery(targetURL string, rawQuery string) string {
	if rawQuery == "" {
		return targetURL
	}
	return targetURL + "?" + rawQuery
}

func inEndpointOf(endpoint string, appInstanceID string, webName string) string {
	return endpoint + PathIn + "/" + appInstanceID + "/" + webName
}
