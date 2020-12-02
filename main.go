package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"index/suffixarray"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	searcher := Searcher{}
	err := searcher.Load("completeworks.txt")
	if err != nil {
		log.Fatal(err)
	}
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/search", handleSearch(searcher))
	http.HandleFunc("/loadmore", handleLoadMore(searcher))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	fmt.Printf("Listening on port %s...", port)
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}

type Searcher struct {
	CompleteWorks string
	SuffixArray   *suffixarray.Index
}

// We can use frameworks to handle http request basic things like middleware, params, response, etc..
func handleLoadMore(searcher Searcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		loadType, ok := r.URL.Query()["type"]
		if !ok || len(loadType[0]) < 1 || (loadType[0] != "prev" && loadType[0] != "nxt") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing search type in URL params"))
			return
		}

		idx, ok := r.URL.Query()["idx"]

		if !ok || len(idx[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing search idx in URL params"))
			return
		}
		ptr, err := strconv.ParseInt(idx[0], 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("idx should be integer"))
			return
		}
		results := searcher.LoadMore(ptr, loadType[0])
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		err = enc.Encode(results)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("encoding failure"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf.Bytes())
	}
}

func handleSearch(searcher Searcher) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query, ok := r.URL.Query()["q"]
		if !ok || len(query[0]) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("missing search query in URL params"))
			return
		}
		results := searcher.Search(query[0])
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		err := enc.Encode(results)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("encoding failure"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(buf.Bytes())
	}
}

func (s *Searcher) Load(filename string) error {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Load: %w", err)
	}
	s.CompleteWorks = string(dat)
	s.SuffixArray = suffixarray.New([]byte(strings.ToLower(s.CompleteWorks)))
	return nil
}

func (s *Searcher) LoadMore(idx int64, loadType string) string {
	return s.CompleteWorks[idx : idx+50]
}

type SearchResult struct {
	Data string
	Prev int
	Next int
}

func (s *Searcher) Search(query string) []SearchResult {
	idxs := s.SuffixArray.Lookup([]byte(strings.ToLower(query)), -1)
	results := []SearchResult{}
	for _, idx := range idxs {
		// We can return the position in another key also
		data := s.CompleteWorks[idx-50:idx] + "<mark>" + s.CompleteWorks[idx:idx+len(query)] + "</mark>" + s.CompleteWorks[idx+len(query):idx+50]
		sr := SearchResult{Data: data, Prev: idx - 50, Next: idx + 50}
		results = append(results, sr)
	}
	return results
}
