package github

import (
	"github.com/google/go-github/github"
)

func getString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func getTimestamp(ts *github.Timestamp) github.Timestamp {
	if ts == nil {
		return github.Timestamp{}
	}
	return *ts
}
