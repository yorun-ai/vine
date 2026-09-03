package http

import "encoding/json/v2"

func unmarshalJson(data []byte, target any) error {
	return json.Unmarshal(data, target)
}
