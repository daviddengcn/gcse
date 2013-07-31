package main

import (
	"encoding/json"
	"unicode/utf8"
	
	"github.com/daviddengcn/go-villa"
)

func JSon(o interface{})[]byte {
	bts, _ := json.Marshal(o)
	return bts
}

func FilterFunc(s string, f func(r rune) bool) string {
	for i, r := range s {
		if f(r) {
			// first time
			buf := villa.ByteSlice(s[:i])
			i += utf8.RuneLen(r)
			for _, r := range s[i:] {
				if !f(r) {
					buf.WriteRune(r)
				}
			}
			return string(buf)
		}
	}
	
	return s
}