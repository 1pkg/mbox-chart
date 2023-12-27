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
	"unicode"

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
		_ = line.Render(w)
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
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
			for !strings.Contains(from, "@") && s.Scan() {
				from = s.Text()
			}
			from = parseEmail(from)
		case strings.HasPrefix(l, "Date:"):
			d := strings.TrimSpace(strings.Replace(l, "Date:", "", 1))
			t, ok := parseDate(d)
			if !ok {
				log.Println("unrecognized format", l)
			}
			date = t
		case strings.HasPrefix(l, "From"):
			if from != "" && !date.IsZero() {
				data[from] = append(data[from], date)
			}
			from, date = "", time.Time{}
		}
	}
	return s.Err()
}

func parseEmail(s string) string {
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
	return string(runes[lo:hi])
}

func parseDate(date string) (time.Time, bool) {
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
			return t, true
		}
	}
	return time.Now(), false
}

type dataset struct {
	year   string
	values []opts.BarData
}

func renderGraph(data map[string][]time.Time) *charts.Bar {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Theme: types.ThemeWonderland}),
		charts.WithDataZoomOpts(opts.DataZoom{Type: "slider"}),
		charts.WithLegendOpts(opts.Legend{Show: true, Type: "scroll", Orient: "horizontal"}),
		charts.WithXAxisOpts(opts.XAxis{
			AxisLabel: &opts.AxisLabel{
				Show:       true,
				Rotate:     90,
				FontSize:   "12",
				FontWeight: "bold",
				Inside:     true,
			},
		}),
	)
	var labels []string
	for k := range data {
		label := strings.ReplaceAll(k, `"`, `\"`)
		labels = append(labels, label)
	}
	var datasets []dataset
	for i, to := 2016, time.Now().Year(); i < to; i++ {
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
			SetSeriesOptions(
				charts.WithBarChartOpts(opts.BarChart{Stack: "stack"}),
				charts.WithItemStyleOpts(opts.ItemStyle{Opacity: 0.75}),
			)
	}
	return bar
}
