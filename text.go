package gcse

import (
	"log"
	"regexp"
	"strings"

	"github.com/golangplus/bytes"
)

/*
	Text analyzes
*/
var patMultiReturns = regexp.MustCompile(`\n\n+`)
var patMultiSpaces = regexp.MustCompile(`\s+`)
var patHeaderBottom = regexp.MustCompile(`((----+)|(====+))\n`)
var patMess = regexp.MustCompile(`((--+==+)|(==+--+))`)

func SplitSentences(text string) []string {
	text = strings.TrimSpace(text)
	text = patHeaderBottom.ReplaceAllString(text, "\n")
	text = patMess.ReplaceAllString(text, "")

	var sents []string
	for _, line := range patMultiReturns.Split(text, -1) {
		line = strings.TrimSpace(line)
		line = strings.Replace(line, "\n", " ", -1)
		rawSents := strings.Split(line, ". ")
		for i, sent := range rawSents {
			if i != len(rawSents)-1 {
				sent = sent + "."
			}
			sent = patMultiSpaces.ReplaceAllString(sent, " ")
			sents = append(sents, sent)
		}
	}

	return sents
}

const (
	reJust = `just`

	reA    = `(an|a|the)`
	reThe  = `(this|the)`
	reThis = `(this|it|these|they)`

	reBasic = `(basic|helper|naive|simple|small|unofficial)`
	reSome  = `((a few)|some|(a set of))`

	reGo = `(pure(-| ))?` + `go` + `(lang|( programming)|(( programming)? language))?`

	reImplementation = `(binding|code|implementation|interface|lib|library|package|plugin|port|program|project|repository|repositorie|tool|utilitie|utility|version|wrapper)`

	reWill         = `(will|would|shall|can|could|(is going to))`
	reImplement    = `(allow|analyze|contain|create|find|implement|perform|provide|wrap)`
	reImplementing = `(allow|analyz|contain|creat|find|implement|perform|provid|wrapp)ing`
	reIs           = `(is|are|be)`

	reOf = `(of|for|to)`

	reVerbSuffix = `(s|ed|d)?`
)

// optional prefix
func op(re string) string {
	return `(` + re + ` )?`
}

// a nount
func n(noun string) string {
	return noun + `s?`
}

// a verb
func v(verb string) string {
	return op(reWill) + verb + reVerbSuffix
}

var (
	isPrefixes = []string{
		`这是一个`,
		`这个项目是`,
		op(reJust) + op(reA) + op(reBasic) + op(reGo) + n(reImplementation) + ` ` + reOf + ` `,
		op(op(reThe)+op(reGo)+n(reImplementation)) + v(reImplement) + ` `,
		op(reGo) + op(reBasic) + n(reImplementation) + ` that `,
		op(reA) + op(reBasic) + reGo + ` `,
		`extended version of ` + reGo + `'s template ` + n(reImplementation) + ` ` + reOf + ` `,
		reA + ` ` + reImplementation + ` ` + reImplementing + ` `,
		reThe + ` ` + reImplementation + ` ` + v(`offer`) + ` ` + reA + ` `,
		reThe + ` ` + reImplementation + ` is designed ` + reOf + ` `,
		reThe + ` ` + reImplementation + ` ` + v(`allow`) + ` ` + reGo + ` `,
		reThe + ` goal of ` + reThe + ` ` + reImplementation + ` is `,
		reThe + ` ` + reImplementation + ` ((can be used)|(allows us)) to `,
		reThe + ` ` + reImplementation + ` ` + reWill + ` `,
		reThis + ` ` + n(reImplement) + ` `,
		op(reThis) + op(reGo) + op(n(reImplementation)) + reIs + ` ` + reA + ` `,
		op(reJust) + reSome + ` ` + n(`function`) + ` ` + reOf + ` `,
		reGo + ` ?的`,
		`api ` + reOf + ` `,
		`to ` + reImplement + ` `,
		`yet another `,
		`solution ` + reOf + ` `,
	}
	isVerbs = []string{
		` ` + op(reJust) + v(reImplement) + ` `,
		` ` + reIs + ` ` + op(`more than`) + reA + ` `,
		` ?是一个`,
		` -+ `,
		`[,:] `,
		` module helps to `,
		` ` + n(`make`) + ` interface ` + reOf + ` `,
		` ` + reImplementation + ` ` + v(reImplement) + ` `,
		` is ` + reGo + ` ` + reImplementation + ` ` + reOf + ` `,
		` ` + reIs + ` supposed to `,
		` ` + reIs + ` intended to `,
	}
	isSuffixes = []string{
		` (for|in) ` + op(reThe) + reGo,
		` using ` + reGo,
	}

	isWrap = [][2]string{
		{reA + ` `, ` ` + reImplementation},
	}

	patISPrefixes = regexp.MustCompile(`^((` + strings.Join(isPrefixes, `)|(`) + `))`)
	patISSuffixes = regexp.MustCompile(`((` + strings.Join(isSuffixes, `)|(`) + `))[.]?\s*(` + reLinkAnchor + `)?$`)
	patISWrap     *regexp.Regexp

	reVerbs = `((` + strings.Join(isVerbs, `)|(`) + `))`

	// ![Build Status](https://secure.travis-ci.org/prometheus/client_golang.png?branch=master)
	reLinkAnchor = `!\[.*?\][(].+?[)]`

	patLinkAnchor *regexp.Regexp = regexp.MustCompile(reLinkAnchor)
)

func init() {
	reWrap := ""
	for _, pair := range isWrap {
		if reWrap != "" {
			reWrap += `|`
		}
		reWrap += `(` + pair[0] + `.+` + pair[1] + `)`
	}
	patISWrap = regexp.MustCompile(`^(` + reWrap + `)[.]?$`)
}

func reEscapeRune(r rune) string {
	switch r {
	case '+', '?', '*', '\\':
		return "[" + string(r) + "]"
	}

	return string(r)
}

func reEscapeString(s string) string {
	if strings.IndexAny(s, "+?*\\") < 0 {
		return s
	}
	var buf bytesp.Slice
	for _, r := range s {
		buf.WriteString(reEscapeRune(r))
	}
	return string(buf)
}

func reName(name string) string {
	var res bytesp.Slice
	p := len(name)
	if strings.HasSuffix(name, "d") || strings.HasSuffix(name, "s") {
		p = len(name) - 1
	} else if strings.HasSuffix(name, "-go") {
		p = len(name) - 3
	}
	for i, r := range name {
		if i > 0 {
			res.WriteString(" ?")
		}
		res.WriteString(reEscapeRune(r))

		if i >= p {
			res.WriteRune('?')
		}
	}
	return string(res)
}

func removeBrackets(sent string) string {
	for {
		l := strings.IndexAny(sent, "(")
		if l < 0 {
			break
		}
		r := strings.IndexAny(sent[l+1:], ")")
		if r < 0 {
			break
		}

		r += l + 1

		if l > 0 && sent[l-1] == byte(' ') && r < len(sent)-1 && sent[r+1] == byte(' ') {
			// if both ends are spaces, increase r to reduce one
			r++
		}
		sent = sent[:l] + sent[r+1:]
	}

	return strings.TrimSpace(sent)
}

func ChooseImportantSentenses(text string, name, pkg string) []string {
	sents := SplitSentences(text)
	if len(sents) < 1 {
		return nil
	}

	name = strings.ToLower(name)

	var namePrefix bytesp.Slice
	namePrefix.WriteString(op(reThe) + `(`)
	parts := strings.Split(strings.ToLower(pkg), "/")
	for i := range parts {
		word := strings.Join(parts[i:], "/")
		if i == len(parts)-1 {
			word = `(go )?` + reName(word) + `([.]go)?`
		} else {
			word = reEscapeString(word)
		}
		re := `(` + word + `)|`
		namePrefix.WriteString(re)
	}

	if name != "" && name != parts[len(parts)-1] {
		namePrefix.WriteString(`((go )?` + reName(name) + `([.]go)?)`)
	}
	namePrefix.WriteString(`)( package)?`)

	re := `^(package )?(` + string(namePrefix) + reVerbs + `)`
	pat, err := regexp.Compile(re)
	if err != nil {
		log.Printf("regexp.Compile %s failed: %v", re, err)
	}

	//log.Printf("pat: %v", pat)

	var importants []string
	for _, sent := range sents {
		lowerSent := removeBrackets(strings.ToLower(sent))
		if patISPrefixes.MatchString(lowerSent) ||
			patISSuffixes.MatchString(lowerSent) ||
			patISWrap.MatchString(lowerSent) ||
			pat != nil && pat.MatchString(lowerSent) {
			sent = patLinkAnchor.ReplaceAllString(sent, "")
			importants = append(importants, sent)
		}
	}
	return importants
}
