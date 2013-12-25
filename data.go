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
	Exported    []string // exported tokens(funcs/types)
}

func (d *DocInfo) WriteTo(w sophie.Writer) error {
	if err := sophie.String(d.Name).WriteTo(w); err != nil {
		return err
	}
	if err := sophie.String(d.Package).WriteTo(w); err != nil {
		return err
	}
	if err := sophie.String(d.Author).WriteTo(w); err != nil {
		return err
	}
	if err := sophie.Time(d.LastUpdated).WriteTo(w); err != nil {
		return err
	}
	if err := sophie.VInt(d.StarCount).WriteTo(w); err != nil {
		return err
	}
	if err := sophie.String(d.Synopsis).WriteTo(w); err != nil {
		return err
	}
	if err := sophie.String(d.Description).WriteTo(w); err != nil {
		return err
	}
	if err := sophie.String(d.ProjectURL).WriteTo(w); err != nil {
		return err
	}
	if err := sophie.String(d.ReadmeFn).WriteTo(w); err != nil {
		return err
	}
	if err := sophie.String(d.ReadmeData).WriteTo(w); err != nil {
		return err
	}
	if err := sophie.WriteStringSlice(w, d.Imports); err != nil {
		return err
	}
	if err := sophie.WriteStringSlice(w, d.Exported); err != nil {
		return err
	}
	return nil
}

func (d *DocInfo) ReadFrom(r sophie.Reader, l int) error {
	if err := (*sophie.String)(&d.Name).ReadFrom(r, -1); err != nil {
		return nil
	}
	if err := (*sophie.String)(&d.Package).ReadFrom(r, -1); err != nil {
		return nil
	}
	if err := (*sophie.String)(&d.Author).ReadFrom(r, -1); err != nil {
		return nil
	}
	if err := (*sophie.Time)(&d.LastUpdated).ReadFrom(r, -1); err != nil {
		return nil
	}
	if err := (*sophie.VInt)(&d.StarCount).ReadFrom(r, -1); err != nil {
		return nil
	}
	if err := (*sophie.String)(&d.Synopsis).ReadFrom(r, -1); err != nil {
		return nil
	}
	if err := (*sophie.String)(&d.Description).ReadFrom(r, -1); err != nil {
		return nil
	}
	if err := (*sophie.String)(&d.ProjectURL).ReadFrom(r, -1); err != nil {
		return nil
	}
	if err := (*sophie.String)(&d.ReadmeFn).ReadFrom(r, -1); err != nil {
		return nil
	}
	if err := (*sophie.String)(&d.ReadmeData).ReadFrom(r, -1); err != nil {
		return nil
	}
	if err := sophie.ReadStringSlice(r, &d.Imports); err != nil {
		return err
	}
	if err := sophie.ReadStringSlice(r, &d.Exported); err != nil {
		return err
	}
	return nil
}

// HitInfo is the information provided to frontend
type HitInfo struct {
	DocInfo

	Imported           []string
	ImportantSentences []string

	StaticScore float64
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
