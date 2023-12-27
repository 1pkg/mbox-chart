package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
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
	cli := influxdb2.NewClient("http://localhost:8086", "QhpqpU8s8WPf9pCwcJMR_YeR51arG8e7QufRaF-rGFlO5BJr_vagES2uX1FrNswx7n12mQ6Gv8UBYMeL6HFTXw==")
	w := cli.WriteAPI("local", "mbox")
	if err := parseMbox(f, w); err != nil {
		log.Fatal(err)
	}
	w.Flush()
	select {
	case err := <-w.Errors():
		log.Fatal(err)
	default:
		log.Print("all done")
	}
}

func parseMbox(r io.Reader, w api.WriteAPI) error {
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
			date = t.Truncate(time.Hour)
		case strings.HasPrefix(l, "From"):
			if from != "" && !date.IsZero() {
				p := influxdb2.NewPoint(
					"mbox",
					map[string]string{"from": from},
					map[string]interface{}{"avg": 1},
					date,
				)
				w.WritePoint(p)
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
