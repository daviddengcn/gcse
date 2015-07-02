package gcse

import (
	"testing"

	"github.com/golangplus/testing/assert"
)

func TestSplitSentences(t *testing.T) {
	TEXT := `
Package gcse is the core supporting library for go-code-search-engine (GCSE).
Its exported types and functions are mainly for sub packages. If you want
some of the function, copy the code away.

Sub-projects

 crawler  crawling packages

indexer  creating index data for web-server 

 --== Godit - a very religious text editor ==--

server   providing web services, including home/top/search services.
`
	SENTS := []string{
		`Package gcse is the core supporting library for go-code-search-engine (GCSE).`,
		`Its exported types and functions are mainly for sub packages.`,
		`If you want some of the function, copy the code away.`,
		`Sub-projects`,
		`crawler crawling packages`,
		`indexer creating index data for web-server`,
		`Godit - a very religious text editor`,
		`server providing web services, including home/top/search services.`,
	}
	sents := SplitSentences(TEXT)
	assert.StringEqual(t, "Sentences", sents, SENTS)
}

func TestChooseImportantSentenses(t *testing.T) {
	TEXT := `
gcse implements something. If you want some of the function, copy the code away.

Package gcse provides something

daviddengcn/core is a something

github/daviddengcn/core is more than a something
-------------------------------------------------
This is a something

gcse是一个something

gcse 是一个something

 is a framework to compare the performance of go 1.0 (go 1.0.3) and go 1.1 (go +tip).

这是一个something

非这是一个something2

the core package provides something

Go language implementation of selected algorithms from the

A simple pluggable lexer package.
`
	IMPORTANTS := []string{
		`gcse implements something.`,
		`Package gcse provides something`,
		`daviddengcn/core is a something`,
		`github/daviddengcn/core is more than a something`,
		`This is a something`,
		`gcse是一个something`,
		`gcse 是一个something`,
		`is a framework to compare the performance of go 1.0 (go 1.0.3) and go 1.1 (go +tip).`,
		`这是一个something`,
		`the core package provides something`,
		`Go language implementation of selected algorithms from the`,
		`A simple pluggable lexer package.`,
	}
	importants := ChooseImportantSentenses(TEXT, "gcse", "github/daviddengcn/core")
	assert.StringEqual(t, "importants", importants, IMPORTANTS)
}

func TestChooseImportantSentenses_GoBot(t *testing.T) {
	TEXT := `
GoBot is an IRC Bot programmed in Golang![Build Status](https://secure.travis-ci.org/prometheus/client_golang.png?branch=master). It is designed to be lightweight and fast.
`
	IMPORTANTS := []string{
		`GoBot is an IRC Bot programmed in Golang.`,
	}
	importants := ChooseImportantSentenses(TEXT, "main", "github.com/wei2912/GoBot")
	assert.StringEqual(t, "importants", importants, IMPORTANTS)
}

func TestChooseImportantSentenses_PackageEscape(t *testing.T) {
	TEXT := `
GoBot is an IRC Bot programmed.
`
	IMPORTANTS := []string{
		`GoBot is an IRC Bot programmed.`,
	}
	importants := ChooseImportantSentenses(TEXT, "main", "github.com/+wei2912/GoBot")
	assert.StringEqual(t, "importants", importants, IMPORTANTS)
}
