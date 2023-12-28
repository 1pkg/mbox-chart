package main

import (
	"bufio"
	"io"
	"strings"
	"time"
	"unicode"
)

type parser struct {
	data map[string][]time.Time
}

func (p parser) Parse(r io.Reader) error {
	var from string
	var date time.Time
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 4096), 1024*1024)
	for s.Scan() {
		l := s.Text()
		switch {
		case strings.HasPrefix(l, "From:"):
			from = strings.Replace(l, "From: ", "", 1)
			for !strings.Contains(from, "@") && s.Scan() {
				from = s.Text()
			}
			from = p.email(from)
		case strings.HasPrefix(l, "Date:"):
			dt := strings.TrimSpace(strings.Replace(l, "Date:", "", 1))
			date = p.time(dt)
		case strings.HasPrefix(l, "From"):
			if from != "" && !date.IsZero() {
				p.data[from] = append(p.data[from], date)
			}
			from, date = "", time.Time{}
		}
	}
	return s.Err()
}

func (parser) email(s string) string {
	runes := []rune(s)
	var at bool
	var lo, hi int = 0, len(runes) - 1
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if unicode.IsSpace(r) || r == rune('<') || r == rune('>') {
			if !at {
				lo = i + 1
			} else {
				hi = i
				break
			}
		}
		at = at || (r == rune('@'))
	}
	return strings.Trim(string(runes[lo:hi]), `'" `)
}

func (parser) time(date string) time.Time {
	date = strings.ReplaceAll(date, ".", ",")
	for _, format := range []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC3339,
		"Mon, 02 Jan 2006 15:04:05 -0700 (MST)",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
		"Mon, 2 Jan 2006 15:04:05 -0700 (GMT+00:00)",
		"Mon, Jan 2, 2006 at 3:04 PM",
		"Mon, Jan 2, 2006 at 04:05",
		"Mon, Jan 2, 2006, 04:05",
		"Mon, 2 May 2006 at 04:05",
		"Mon 2, 1, 2006 at 04:05",
		"02 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 -0700",
		"1/2/2006",
	} {
		t, err := time.Parse(format, date)
		if err == nil {
			return t
		}
	}
	return time.Now()
}
