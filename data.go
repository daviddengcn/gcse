package gcse

import (
	"encoding/gob"
	"github.com/agonopol/go-stem/stemmer"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// DocInfo is the information stored in backend docDB
type DocInfo struct {
	Name        string
	Package     string
	Author      string
	LastUpdated time.Time
	StarCount   int
	Synopsis    string
	Description string
	ProjectURL  string
	ReadmeFn    string
	ReadmeData  string
	Imports     []string
}

// HitInfo is the information provided to frontend
type HitInfo struct {
	DocInfo
	Imported []string

	StaticScore float64
}

func init() {
	gob.Register(DocInfo{})
	gob.Register(HitInfo{})
}

var patURL = regexp.MustCompile(`http[s]?://\S+`)

func filterURLs(text string) string {
	return patURL.ReplaceAllString(text, " ")
}

func isTermSep(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r)
}

var stemBlackList = map[string]string{
	"ide":      "ide",
	"generics": "generic",
	"generic":  "generic",
}

func NormWord(word string) string {
	word = strings.ToLower(word)
	if mapWord, ok := stemBlackList[word]; ok {
		word = mapWord
	} else {
		word = string(stemmer.Stem([]byte(word)))
	}
	return word
}

var stopWords = villa.NewStrSet([]string{
	"the", "on", "in", "as",
}...)

func CheckRuneType(last, current rune) index.RuneType {
	if isTermSep(current) {
		return index.TokenSep
	}

	if current > 128 {
		return index.TokenStart
	}

	if unicode.IsLetter(current) {
		if unicode.IsLetter(last) {
			return index.TokenBody
		}
		return index.TokenStart
	}

	if unicode.IsNumber(current) {
		if unicode.IsNumber(last) {
			return index.TokenBody
		}
		return index.TokenStart
	}

	return index.TokenStart
}

func isCamel(token string) bool {
	upper, lower := false, false
	for _, r := range token {
		if !unicode.IsLetter(r) {
			return false
		}

		if unicode.IsUpper(r) {
			upper = true
			if lower {
				break
			}
		} else {
			lower = true
		}
	}

	return upper && lower
}

func CheckCamel(last, current rune) index.RuneType {
	if unicode.IsUpper(current) {
		return index.TokenStart
	}

	return index.TokenBody
}

func AppendTokens(tokens villa.StrSet, text string) villa.StrSet {
	text = filterURLs(text)

	lastToken := ""
	index.Tokenize(CheckRuneType, villa.NewPByteSlice([]byte(text)), func(token []byte) error {
		tokenStr := string(token)
		if isCamel(tokenStr) {
			last := ""
			index.Tokenize(CheckCamel, villa.NewPByteSlice(token), func(token []byte) error {
				tokenStr = string(token)
				tokenStr = NormWord(tokenStr)
				if !stopWords.In(tokenStr) {
					tokens.Put(tokenStr)
				}

				if last != "" {
					tokens.Put(last + "-" + string(tokenStr))
				}

				last = tokenStr
				return nil
			})
		}
		tokenStr = NormWord(tokenStr)
		if !stopWords.In(tokenStr) {
			tokens.Put(tokenStr)
		}

		if tokenStr[0] > 128 && len(lastToken) > 0 && lastToken[0] > 128 {
			tokens.Put(lastToken + tokenStr)
		}

		lastToken = tokenStr
		return nil
	})

	return tokens
}
