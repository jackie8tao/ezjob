package jsonutil

import (
	"encoding/json"
)

func ToBytes(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return data
}
