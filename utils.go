package gcse

import (
	"encoding/json"
	"github.com/daviddengcn/go-villa"
)

func WriteJsonFile(fn villa.Path, data interface{}) error {
	f, err := fn.Create()
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(data)
}

func ReadJsonFile(fn villa.Path, data interface{}) error {
	f, err := fn.Open()
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	return dec.Decode(data)
}
