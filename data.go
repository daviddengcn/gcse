package gcse

import (
	"encoding/gob"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/golangplus/bytes"
	"github.com/golangplus/errors"
	"github.com/golangplus/strings"

	"github.com/agonopol/go-stem"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/sophie"
)

// DocInfo is the information stored in backend docDB
type DocInfo struct {
	Name        string // Package name
	Package     string // Package path
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

// Returns a new instance of DocInfo as a sophie.Sophier
func NewDocInfo() sophie.Sophier {
	return new(DocInfo)
}

func (d *DocInfo) WriteTo(w sophie.Writer) error {
	return errorsp.WithStacks(gob.NewEncoder(w).Encode(d))
}

func (d *DocInfo) ReadFrom(r sophie.Reader, l int) error {
	// clear before decoding, otherwise some slice will be reused
	*d = DocInfo{}
	return errorsp.WithStacks(gob.NewDecoder(r).Decode(d))
}

// HitInfo is the information provided to frontend
type HitInfo struct {
	DocInfo

	Imported    []string
	ImportedLen int

	TestImported    []string
	TestImportedLen int

	ImportantSentences []string

	AssignedStarCount float64
	StaticScore       float64
	TestStaticScore   float64
	StaticRank        int // zero-based
}

func init() {
	gob.Register(DocInfo{})
	gob.Register(HitInfo{})
}

var patURL = regexp.MustCompile(`http[s]?://\S+`)

func filterURLs(text []byte) []byte {
	return patURL.ReplaceAll(text, []byte(" "))
}

var patEmail = regexp.MustCompile(`[A-Za-z0-9_.+-]+@([a-zA-Z0-9_-]+[.])+[A-Za-z]+`)

func filterEmails(text []byte) []byte {
	return patEmail.ReplaceAll(text, nil)
}

func isTermSep(r rune) bool {
	return unicode.IsPunct(r) || unicode.IsSymbol(r) || r == 0xfeff
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

var stopWords = stringsp.NewSet(
	"the", "on", "in", "as",
)

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
func appendTokensOfBlock(tokens stringsp.Set, block []byte) stringsp.Set {
	lastToken := ""
	index.Tokenize(CheckRuneType, (*bytesp.Slice)(&block),
		func(token []byte) error {
			tokenStr := string(token)
			if isCamel(tokenStr) {
				last := ""
				index.Tokenize(CheckCamel, bytesp.NewPSlice(token),
					func(token []byte) error {
						tokenStr := string(token)
						tokenStr = NormWord(tokenStr)
						if !stopWords.Contain(tokenStr) {
							tokens.Add(tokenStr)
						}
						if last != "" {
							tokens.Add(last + string(tokenStr))
						}
						last = tokenStr
						return nil
					})
			}
			tokenStr = NormWord(tokenStr)
			if !stopWords.Contain(tokenStr) {
				tokens.Add(tokenStr)
			}
			if lastToken != "" {
				if tokenStr[0] > 128 && lastToken[0] > 128 {
					// Chinese bigrams
					tokens.Add(lastToken + tokenStr)
				} else if tokenStr[0] <= 128 && lastToken[0] <= 128 {
					tokens.Add(lastToken + "-" + tokenStr)
				}
			}
			lastToken = tokenStr
			return nil
		})
	return tokens
}

// Tokenizes text into the current token set.
func AppendTokens(tokens stringsp.Set, text []byte) stringsp.Set {
	textBuf := filterURLs(text)
	textBuf = filterEmails(textBuf)

	index.Tokenize(index.SeparatorFRuneTypeFunc(unicode.IsSpace),
		(*bytesp.Slice)(&textBuf), func(block []byte) error {
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
