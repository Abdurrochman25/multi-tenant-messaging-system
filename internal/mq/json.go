package mq

import "encoding/json"

func ParseJSON[T any](b []byte) (T, bool) {
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return v, false
	}
	return v, true
}
