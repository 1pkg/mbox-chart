package main

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
)

type plot struct {
	data map[string][]time.Time
}

func (p plot) Render(w io.Writer) error {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Theme: types.ThemeWonderland,
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type: "slider",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: true, Type: "scroll",
			Orient: "horizontal",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			AxisLabel: &opts.AxisLabel{
				Show:       true,
				Rotate:     90,
				FontSize:   "14",
				FontWeight: "bold",
				Inside:     true,
				Interval:   "0",
			},
		}),
	)
	labels, data := p.dataset()
	bar.SetXAxis(labels)
	min, max := p.minYear(), time.Now().Year()
	for i, to := min, max; i <= to; i++ {
		bar.AddSeries(fmt.Sprint(i), data[i]).SetSeriesOptions(
			charts.WithBarChartOpts(opts.BarChart{Stack: "stack"}),
			charts.WithItemStyleOpts(opts.ItemStyle{Opacity: 0.5}),
			charts.WithLabelOpts(opts.Label{Show: true, Position: "right"}),
		)
	}
	return bar.Render(w)
}

func (p plot) minYear() int {
	var min time.Time
	for _, times := range p.data {
		for _, t := range times {
			if min.IsZero() || t.Before(min) {
				min = t
			}
		}
	}
	return min.Year()
}

func (p plot) dataset() (labels []string, data map[int][]opts.BarData) {
	for l := range p.data {
		labels = append(labels, l)
	}
	totals := make(map[string]int)
	for l, times := range p.data {
		for range times {
			totals[l] += 1
		}
	}
	sort.Slice(labels, func(i, j int) bool {
		return totals[labels[i]] < totals[labels[j]]
	})
	data = make(map[int][]opts.BarData)
	min, max := p.minYear(), time.Now().Year()
	for _, l := range labels {
		times := p.data[l]
		if len(times) == 0 {
			continue
		}
		for i, to := min, max; i <= to; i++ {
			var total int
			for _, t := range times {
				if t.Year() == i {
					total++
				}
			}
			data[i] = append(data[i], opts.BarData{Value: total})
		}
	}
	return
}
