package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"
)

const horizont = 90 * 24 * time.Hour

//go:embed index.html.tmpl
var itmpl string

type Dataset struct {
	Label  string
	Values []int
}

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
	if len(data) == 0 {
		log.Fatal("no data in mbox")
	}
	var key string
	for k := range data {
		sort.Slice(data[k], func(i, j int) bool {
			return data[k][i].Before(data[k][j])
		})
		key = k
	}
	min, max := data[key][0], data[key][len(data[key])-1]
	for k := range data {
		if m := data[k][0]; m.Before(min) {
			min = m
		}
		if m := data[key][0]; m.After(max) {
			max = m
		}
	}
	if err := renderGraph(data, min, max); err != nil {
		log.Fatal(err)
	}
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
	buf := make([]byte, 0, 64*1024)
	s.Buffer(buf, 1024*1024)
	for s.Scan() {
		l := s.Text()
		switch {
		case strings.HasPrefix(l, "From:"):
			from = strings.Replace(l, "From: ", "", 1)
			if !strings.Contains(from, "@") {
				for s.Scan() {
					l := s.Text()
					if strings.Contains(l, "@") {
						from = l
						break
					}
				}
			}
		case strings.HasPrefix(l, "Date:"):
			t := parseDate(strings.Replace(l, "Date: ", "", 1))
			if t == nil {
				log.Println("unrecognized format", l)
				continue
			}
			date = t.Truncate(horizont)
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

func renderGraph(data map[string][]time.Time, min, max time.Time) error {
	f, err := os.Create("index.html")
	if err != nil {
		return err
	}
	tmpl, err := template.New("index").Parse(itmpl)
	if err != nil {
		return err
	}
	var labels []string
	for t := min; t.Before(max); t = t.Add(horizont) {
		labels = append(labels, t.Format(time.DateOnly))
	}
	var datasets []Dataset
	for k := range data {
		d := data[k]
		ds := Dataset{Label: fmt.Sprintf("%s (%d)", strings.ReplaceAll(k, `"`, `\"`), len(d))}
		var ptr int
		for i := 0; i < len(labels); i++ {
			l := labels[i]
			var j int = ptr
			for ; ptr < len(d); ptr++ {
				if l != d[ptr].Format(time.DateOnly) {
					break
				}
			}
			ds.Values = append(ds.Values, ptr-j)
		}
		datasets = append(datasets, ds)
	}
	return tmpl.Execute(f, struct {
		Labels   []string
		Datasets []Dataset
	}{Labels: labels, Datasets: datasets})
}
