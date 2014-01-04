package main

import (
	"github.com/daviddengcn/gcse"
	"github.com/daviddengcn/go-index"
	"github.com/daviddengcn/go-villa"
	"log"
	"runtime"
	"strings"
	"time"
)

type Hit struct {
	gcse.HitInfo
	MatchScore float64
	Score      float64
}

type SearchResult struct {
	TotalResults int
	Hits         []*Hit
}

var stopWords = villa.NewStrSet([]string{
	"the", "on", "in", "as",
}...)

var (
	indexDBBox   villa.AtomicBox
	indexSegment gcse.Segment
	indexUpdated time.Time
)

func loadIndex() error {
	segm, err := gcse.IndexSegments.FindMaxDone()
	if segm == nil || err != nil {
		return err
	}

	if indexSegment != nil && !gcse.SegmentLess(indexSegment, segm) {
		// no new index
		return nil
	}

	db := &index.TokenSetSearcher{}
	f, err := segm.Join(gcse.IndexFn).Open()
	if err != nil {
		return err
	}
	defer f.Close()

	if err := db.Load(f); err != nil {
		return err
	}

	indexSegment = segm
	log.Printf("Load index from %v (%d packages)", segm, db.DocCount())

	indexDBBox.Set(db)
	updateTime := time.Now()

	if st, err := segm.Join(gcse.IndexFn).Stat(); err == nil {
		updateTime = st.ModTime()
	}

	indexUpdated = updateTime

	db = nil
	gcse.DumpMemStats()
	runtime.GC()
	gcse.DumpMemStats()

	return nil
}

func loadIndexLoop() {
	for {
		time.Sleep(30 * time.Second)

		if err := loadIndex(); err != nil {
			log.Printf("loadIndex failed: %v", err)
		}
	}
}

func search(q string) (*SearchResult, villa.StrSet, error) {
	tokens := gcse.AppendTokens(nil, []byte(q))
	log.Printf("tokens for query %s: %v", q, tokens)

	indexDB := indexDBBox.Get().(*index.TokenSetSearcher)

	if indexDB == nil {
		return &SearchResult{}, tokens, nil
	}

	var hits []*Hit

	N := indexDB.DocCount()
	Df := func(token string) int {
		return len(indexDB.TokenDocList(gcse.IndexTextField, token))
	}

	_, _ = N, Df

	indexDB.Search(map[string]villa.StrSet{gcse.IndexTextField: tokens},
		func(docID int32, data interface{}) error {
			hitInfo, _ := data.(gcse.HitInfo)
			hit := &Hit{
				HitInfo: hitInfo,
			}

			hit.MatchScore = gcse.CalcMatchScore(&hitInfo, tokens, N, Df)
			hit.Score = hit.StaticScore * hit.MatchScore

			hits = append(hits, hit)
			return nil
		})

	log.Printf("Got %d hits for query %q", len(hits), q)

	villa.SortF(len(hits), func(i, j int) bool {
		// true if doc i is before doc j
		ssi, ssj := hits[i].Score, hits[j].Score
		if ssi > ssj {
			return true
		}
		if ssi < ssj {
			return false
		}

		sci, scj := hits[i].StarCount, hits[j].StarCount
		if sci > scj {
			return true
		}
		if sci < scj {
			return false
		}

		pi, pj := hits[i].Package, hits[j].Package
		if len(pi) < len(pj) {
			return true
		}
		if len(pi) > len(pj) {
			return false
		}

		return pi < pj
	}, func(i, j int) {
		// Swap
		hits[i], hits[j] = hits[j], hits[i]
	})

	return &SearchResult{
		TotalResults: len(hits),
		Hits:         hits,
	}, tokens, nil
}

func splitToLines(text string) []string {
	lines := strings.Split(text, "\n")
	newLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		newLines = append(newLines, line)
	}

	return newLines
}

func selectSnippets(text string, tokens villa.StrSet, maxBytes int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxBytes {
		return text
	}
	// return text[:maxBytes] + "..."

	lines := splitToLines(text)

	var hitTokens villa.StrSet
	type lineinfo struct {
		idx  int
		line string
	}
	var selLines []lineinfo
	count := 0
	for i, line := range lines {
		line = strings.TrimSpace(line)
		lines[i] = line

		lineTokens := gcse.AppendTokens(nil, []byte(line))
		reserve := false
		for token := range tokens {
			if !hitTokens.In(token) && lineTokens.In(token) {
				reserve = true
				hitTokens.Put(token)
			}
		}

		if i == 0 || reserve && (count+len(line)+1 < maxBytes) {
			selLines = append(selLines, lineinfo{
				idx:  i,
				line: line,
			})
			count += len(line) + 1
			if count == maxBytes {
				break
			}

			lines[i] = ""
		}
	}

	if count < maxBytes {
		for i, line := range lines {
			if len(line) == 0 {
				continue
			}

			if count+len(line) >= maxBytes {
				break
			}

			selLines = append(selLines, lineinfo{
				idx:  i,
				line: line,
			})

			count += len(line) + 1
		}

		villa.SortF(len(selLines), func(i, j int) bool {
			return selLines[i].idx < selLines[j].idx
		}, func(i, j int) {
			selLines[i], selLines[j] = selLines[j], selLines[i]
		})
	}

	var outBuf villa.ByteSlice
	for i, line := range selLines {
		if line.idx > 1 && (i < 1 || line.idx != selLines[i-1].idx+1) {
			outBuf.WriteString("...")
		} else {
			if i > 0 {
				outBuf.WriteString(" ")
			}
		}
		outBuf.WriteString(line.line)
	}

	if selLines[len(selLines)-1].idx != len(lines)-1 {
		outBuf.WriteString("...")
	}

	return string(outBuf)
}
