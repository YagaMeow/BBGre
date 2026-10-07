package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// LyricLine is a single timed lyric line.
type LyricLine struct {
	Time    float64 `json:"time"`
	Content string  `json:"content"`
}

var (
	lrcTimeRe = regexp.MustCompile(`\[(\d{1,3}):(\d{1,2})(?:[.:](\d{1,3}))?\]`)
	lrcMetaRe = regexp.MustCompile(`^\[(ti|ar|al|by|offset|re|ve):(.*)\]$`)
)

// ParseLyrics understands three common lyric formats:
//   - standard LRC lines: [mm:ss.xx]text
//   - LRC metadata tags:  [ti:...], [offset:...]
//   - NetEase JSON lines: {"t":1234,"c":[{"tx":"text"}]}
//
// The result is sorted by time and normalised to seconds.
func ParseLyrics(content string) []LyricLine {
	lines := make([]LyricLine, 0, 64)
	offset := 0.0

	sc := bufio.NewScanner(strings.NewReader(content))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		raw := strings.TrimSpace(sc.Text())
		if raw == "" {
			continue
		}

		if strings.HasPrefix(raw, "{") {
			if line, ok := parseNeteaseLine(raw, offset); ok {
				lines = append(lines, line)
			}
			continue
		}

		if m := lrcMetaRe.FindStringSubmatch(raw); m != nil {
			if strings.ToLower(m[1]) == "offset" {
				if v, err := strconv.ParseFloat(strings.TrimSpace(m[2]), 64); err == nil {
					offset = v / 1000.0
				}
			}
			continue
		}

		matches := lrcTimeRe.FindAllStringSubmatch(raw, -1)
		if len(matches) == 0 {
			continue
		}
		text := strings.TrimSpace(lrcTimeRe.ReplaceAllString(raw, ""))
		for _, tm := range matches {
			lines = append(lines, LyricLine{
				Time:    lrcTimeToSeconds(tm) + offset,
				Content: text,
			})
		}
	}

	sort.SliceStable(lines, func(i, j int) bool { return lines[i].Time < lines[j].Time })
	for i := range lines {
		if lines[i].Time < 0 {
			lines[i].Time = 0
		}
	}
	return lines
}

func lrcTimeToSeconds(m []string) float64 {
	min, _ := strconv.Atoi(m[1])
	sec, _ := strconv.Atoi(m[2])
	frac := 0.0
	switch len(m[3]) {
	case 1:
		frac = float64(atoi(m[3])) / 10
	case 2:
		frac = float64(atoi(m[3])) / 100
	case 3:
		frac = float64(atoi(m[3])) / 1000
	}
	return float64(min*60+sec) + frac
}

func atoi(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func parseNeteaseLine(raw string, offset float64) (LyricLine, bool) {
	var payload struct {
		T int64 `json:"t"`
		C []struct {
			Tx string `json:"tx"`
		} `json:"c"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return LyricLine{}, false
	}
	if len(payload.C) == 0 {
		return LyricLine{}, false
	}
	var segment strings.Builder
	for _, c := range payload.C {
		segment.WriteString(c.Tx)
	}
	text := strings.TrimSpace(segment.String())
	if text == "" {
		return LyricLine{}, false
	}
	t := float64(payload.T)/1000.0 + offset
	if t < 0 {
		t = 0
	}
	return LyricLine{Time: t, Content: text}, true
}

// LyricsToLRC serialises parsed lyric lines back into the standard LRC format
// so the frontend can fetch them as a plain .lrc file.
func LyricsToLRC(lines []LyricLine) string {
	var b strings.Builder
	for _, l := range lines {
		if l.Content == "" {
			continue
		}
		total := l.Time
		if total < 0 {
			total = 0
		}
		min := int(total) / 60
		sec := int(total) % 60
		csec := int((total - float64(int(total))) * 100)
		fmt.Fprintf(&b, "[%02d:%02d.%02d]%s\n", min, sec, csec, l.Content)
	}
	return b.String()
}