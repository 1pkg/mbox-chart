package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	f, err := readFile()
	if err != nil {
		log.Fatal(err)
	}
	if f == nil {
		log.Fatal("mbox is required")
	}
	defer f.Close()
	data := make(map[string][]time.Time)
	if err := parseMbox(f, data); err != nil {
		log.Fatal(err)
	}
	for k := range data {
		sort.Slice(data[k], func(i, j int) bool {
			return data[k][i].Before(data[k][j])
		})
	}
	fmt.Println(data)
}

func readFile() (io.ReadCloser, error) {
	if len(os.Args) != 2 {
		return nil, nil
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		return nil, err
	}
	return f, nil
}

func parseMbox(r io.Reader, data map[string][]time.Time) error {
	var from string
	var date time.Time
	s := bufio.NewScanner(r)
	for s.Scan() {
		l := s.Text()
		switch {
		case strings.HasPrefix(l, "From:"):
			from = strings.Replace(l, "From: ", "", 1)
		case strings.HasPrefix(l, "Date:"):
			t := parseDate(strings.Replace(l, "Date: ", "", 1))
			if t == nil {
				log.Println("unrecognized format", l)
				continue
			}
			date = t.Round(time.Hour)
		case strings.HasPrefix(l, "From"):
			if from != "" && !date.IsZero() {
				data[from] = append(data[from], date)
			}
			from, date = "", time.Time{}
		}
	}
	return s.Err()
}

func parseDate(date string) *time.Time {
	for _, format := range []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 02 Jan 2006 15:04:05 -0700 (MST)",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		time.RFC3339,
	} {
		t, err := time.Parse(format, date)
		if err == nil {
			return &t
		}
	}
	return nil
}
