package main

import (
	"log"
	"net/http"
	"os"
	"time"
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
	p := parser{data: make(map[string][]time.Time)}
	if err := p.Parse(f); err != nil {
		log.Fatal(err)
	}
	if len(p.data) == 0 {
		log.Fatal("no data in mbox")
	}
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		p := plot(p)
		_ = p.Render(w)
	})
	log.Println("http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
