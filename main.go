package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
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

	var data []byte
	switch pathType {
	case "file":
		var file *os.File
		file, res.err = os.Open(res.path)
		if res.err != nil {
			return res
		}
		defer file.Close()

		data, res.err = ioutil.ReadAll(file)
	case "url":
		var resp *http.Response
		resp, res.err = http.Get(path)
		if res.err != nil {
			return res
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			res.err = fmt.Errorf("response: %s", resp.Status)
			return res
		}

		data, res.err = ioutil.ReadAll(resp.Body)
	default:
		res.err = fmt.Errorf("pathType: type %s isn't supported", pathType)
	}

	if res.err == nil {
		res.count = bytes.Count(data, []byte("Go"))
	}

	return res
}

func main() {
	help := "gocounter \"input\" -type url|file. "
	var input []byte
	var pathType = flag.String("type", "", "Specifies type of incoming pathes: file or url")
	flag.Parse()

	stat, err := os.Stdin.Stat()
	if err != nil {
		log.Fatalf("Stdin.Stat(): %s", err)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Read from stdin
		input, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Failed to read from stdin: %s", err)
		}
	} else {
		// Read from console otherwise
		if len(os.Args) < 4 {
			fmt.Println(help)
			return
		}
		input = []byte(os.Args[1])
		*pathType = os.Args[3]
	}

	if *pathType != "file" && *pathType != "url" {
		fmt.Println(help)
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

	// Processes links
	var wg sync.WaitGroup
	goroutines := make(chan struct{}, 5)
	for _, path := range bytes.Split(input, []byte("\n")) {
		//for _, path := range strings.Split(input, "\n") {
		if len(path) == 0 {
			continue
		}

		goroutines <- struct{}{}
		wg.Add(1)
		go func(path, pathType string, goroutines <-chan struct{}, results chan<- result, wg *sync.WaitGroup) {
			results <- process(path, pathType)
			<-goroutines
			wg.Done()
		}(string(path), *pathType, goroutines, results, &wg)
	}

	wg.Wait()
	close(goroutines)
	close(results)
	<-done
}
