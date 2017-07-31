package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

type result struct {
	path     string
	pathType string
	count    int
	err      error
}

func process(path, pathType string) result {
	var res result
	if path == "" {
		res.err = errors.New("process: empty path")
		return res
	}

	res = result{
		path:     path,
		pathType: pathType,
	}

	var reader io.ReadCloser
	switch pathType {
	case "file":
		reader, res.err = os.Open(res.path)
	case "url":
		var resp *http.Response
		resp, res.err = http.Get(path)
		if res.err != nil {
			return res
		}

		if resp.StatusCode != http.StatusOK {
			res.err = fmt.Errorf("response: %s", resp.Status)
		}
		reader = resp.Body
	default:
		res.err = fmt.Errorf("pathType: type %s isn't supported", pathType)
	}

	if res.err == nil {
		defer reader.Close()
		p := make([]byte, 2)
		reader := bufio.NewReader(reader)
		_, err := reader.Read(p)
		for err == nil {
			if bytes.Equal(p, []byte("Go")) {
				res.count = res.count + 1
			}
			p[0] = p[1]
			p[1], err = reader.ReadByte()
		}
	}

	return res
}

func main() {
	var pathType = flag.String("type", "", "Specifies type of incoming pathes: file or url")
	flag.Parse()

	stat, err := os.Stdin.Stat()
	if err != nil {
		log.Fatalf("Stdin.Stat(): %s", err)
	}

	if (stat.Mode() & os.ModeCharDevice) != 0 {
		log.Fatal("gocounter accepts data only from stdin")
	}

	if *pathType != "file" && *pathType != "url" {
		fmt.Println("gocounter -type url|file. ")
		return
	}

	results := make(chan result)

	// Reads results and print to console
	done := make(chan struct{})
	go func(results <-chan result, done chan<- struct{}) {
		var total int
		for result := range results {
			if result.err != nil {
				log.Printf("error: %v path: %s type: %s", result.err, result.path, result.pathType)
				continue
			}
			fmt.Printf("Count for %s: %d\n", result.path, result.count)
			total += result.count
		}
		fmt.Printf("Total: %d\n", total)
		done <- struct{}{}
	}(results, done)

	scanner := bufio.NewScanner(os.Stdin)
	// Processes links
	var wg sync.WaitGroup
	goroutines := make(chan struct{}, 5)
	for scanner.Scan() {
		path := scanner.Text()
		goroutines <- struct{}{}
		wg.Add(1)
		go func(path, pathType string, goroutines <-chan struct{}, results chan<- result, wg *sync.WaitGroup) {
			results <- process(path, pathType)
			<-goroutines
			wg.Done()
		}(path, *pathType, goroutines, results, &wg)
	}

	wg.Wait()
	close(goroutines)
	close(results)
	<-done
}
