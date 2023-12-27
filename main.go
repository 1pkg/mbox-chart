package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("mbox filepath is required")
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	data := make(map[string][]time.Time)
	if err := parseMbox(f, data); err != nil {
		log.Fatal(err)
	}
	if len(data) == 0 {
		log.Fatal("no data in mbox")
	}
	for k := range data {
		sort.Slice(data[k], func(i, j int) bool {
			return data[k][i].Before(data[k][j])
		})
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		line := renderGraph(data)
		line.Render(w)
	})
	http.ListenAndServe(":8081", nil)
}

func parseMbox(r io.Reader, data map[string][]time.Time) error {
	var from string
	var date time.Time
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 4096), 1024*1024)
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
			date = *t
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

type dataset struct {
	year   string
	values []opts.BarData
}

func renderGraph(data map[string][]time.Time) *charts.Bar {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWesteros}),
		charts.WithDataZoomOpts(opts.DataZoom{}),
		charts.WithLegendOpts(opts.Legend{Show: true, Type: "scroll", Orient: "horizontal"}),
	)
	var labels []string
	for k := range data {
		label := strings.ReplaceAll(k, `"`, `\"`)
		labels = append(labels, label)
	}
	var datasets []dataset
	for i := 2016; i < 2024; i++ {
		ds := dataset{year: fmt.Sprint(i)}
		for _, times := range data {
			if len(times) == 0 {
				continue
			}
			var total int
			for _, t := range times {
				if t.Year() == i {
					total++
				}
			}
			ds.values = append(ds.values, opts.BarData{Value: total})
		}
		datasets = append(datasets, ds)
	}
	bar.SetXAxis(labels)
	for _, ds := range datasets {
		bar.AddSeries(ds.year, ds.values).
			SetSeriesOptions(charts.WithBarChartOpts(opts.BarChart{
				Stack: "stack",
			}))
	}
	return bar
}
