package utils

import (
	"encoding/json"
	"os"
)

func WriteJsonFile(fn string, data interface{}) error {
	f, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(data)
}

func ReadJsonFile(fn string, data interface{}) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	return dec.Decode(data)
}
