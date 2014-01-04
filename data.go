package gcse

import (
	"encoding/gob"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/agonopol/go-stem/stemmer"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"github.com/daviddengcn/sophie"
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
	TestImports []string
	Exported    []string // exported tokens(funcs/types)
}

func (d *DocInfo) WriteTo(w sophie.Writer) error {
	return gob.NewEncoder(w).Encode(d)
}

func (d *DocInfo) ReadFrom(r sophie.Reader, l int) error {
	// clear before decoding, otherwise some slice will be reused
	*d = DocInfo{}
	return gob.NewDecoder(r).Decode(d)
}

// HitInfo is the information provided to frontend
type HitInfo struct {
	DocInfo

	Imported           []string
	TestImported       []string
	ImportantSentences []string

	StaticScore float64
	TestStaticScore float64
	StaticRank  int // zero-based
}

func init() {
	gob.Register(DocInfo{})
	gob.Register(HitInfo{})
}

var patURL = regexp.MustCompile(`http[s]?://\S+`)

func filterURLs(text []byte) []byte {
	return patURL.ReplaceAll(text, []byte(" "))
}

func isTermSep(r rune) bool {
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
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

// a block does not contain blanks
func appendTokensOfBlock(tokens villa.StrSet, block []byte) villa.StrSet {
	lastToken := ""
	index.Tokenize(CheckRuneType, (*villa.ByteSlice)(&block),
		func(token []byte) error {
			tokenStr := string(token)
			if isCamel(tokenStr) {
				last := ""
				index.Tokenize(CheckCamel, villa.NewPByteSlice(token),
					func(token []byte) error {
						tokenStr := string(token)
						tokenStr = NormWord(tokenStr)
						if !stopWords.In(tokenStr) {
							tokens.Put(tokenStr)
						}

						if last != "" {
							tokens.Put(last + string(tokenStr))
						}

						last = tokenStr
						return nil
					})
			}
			tokenStr = NormWord(tokenStr)
			if !stopWords.In(tokenStr) {
				tokens.Put(tokenStr)
			}

			if lastToken != "" {
				if tokenStr[0] > 128 && lastToken[0] > 128 {
					// Chinese bigrams
					tokens.Put(lastToken + tokenStr)
				} else if tokenStr[0] <= 128 && lastToken[0] <= 128 {
					tokens.Put(lastToken + "-" + tokenStr)
				}
			}

			lastToken = tokenStr
			return nil
		})
	return tokens
}

func AppendTokens(tokens villa.StrSet, text []byte) villa.StrSet {
	textBuf := filterURLs([]byte(text))

	index.Tokenize(index.SeparatorFRuneTypeFunc(unicode.IsSpace),
		(*villa.ByteSlice)(&textBuf), func(block []byte) error {
			tokens = appendTokensOfBlock(tokens, block)
			return nil
		})

	return tokens
}

const (
	DOCS_PARTS = 128
)

func CalcPackagePartition(pkg string, totalParts int) int {
	hash := 0
	for i, l := 0, len(pkg); i < l; i++ {
		b := pkg[i]
		hash = hash*33 + int(b)
		if hash > totalParts {
			hash = hash % totalParts
		}
	}

	return hash
}
