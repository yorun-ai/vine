package meta

import (
	"encoding/base64"

	"go.yorun.ai/vine/util/vcode"
)

func decodePayloadFromBase64[T any](value string) (T, error) {
	jsonBytes, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return *new(T), err
	}

	payload, err := vcode.UnmarshalJson[T](jsonBytes)
	if err != nil {
		return *new(T), err
	}
	return payload, nil
}

func encodePayloadToBase64(payload any) string {
	return base64.RawURLEncoding.EncodeToString(vcode.MustMarshalJson(payload))
}
