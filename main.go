package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

type result struct {
	path     string
	pathType string
	count    int
	err      error
}

// TODO: Write tests
// Returns amount of word "Go" in string
func goCounter(data string) (count int, err error) {
	re, err := regexp.Compile("\\WGo\\W")
	if err != nil {
		return
	}
	count = len(re.FindAllString(data, -1))
	return
}

// TODO: Write tests
func process(path, pathType string) result {
	var res result
	if path == "" {
		res.err = errors.New("path: invalid path")
		return res
	}
	res.path = path
	res.pathType = pathType
	switch pathType {
	case "file":
		file, err := os.Open(res.path)
		if err != nil {
			res.err = err
			return res
		}
		defer file.Close()
		data, err := ioutil.ReadAll(file)
		if err != nil {
			res.err = err
			return res
		}
		res.count, res.err = goCounter(string(data))
	case "url":
		resp, err := http.Get(path)
		if err != nil {
			res.err = err
			return res
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			res.err = err
			return res
		}
		res.count, res.err = goCounter(string(body))
	default:
		res.err = errors.New("pathType: unknow type")
	}
	return res
}

func main() {
	var input string
	var pathType = flag.String("type", "", "Specifies type of incoming pathes: file or url")
	flag.Parse()

	stat, err := os.Stdin.Stat()
	if err != nil {
		log.Fatalf("Stdin.Stat(): %s", err)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Read from stdin
		bytes, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Failed to read from stdin: %s", err)
		}
		input = string(bytes)
	} else {
		// Read from console otherwise
		if len(os.Args) < 4 {
			fmt.Println("gocounter \"input\" -type url|file")
			return
		}
		input = os.Args[1]
		*pathType = os.Args[3]
	}

	// Reads results and print to console
	done := make(chan struct{})
	results := make(chan result)
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
	for _, path := range strings.Split(input, "\n") {
		if path == "" {
			continue
		}

		goroutines <- struct{}{}
		wg.Add(1)
		go func(path, pathType string, goroutines <-chan struct{}, wg *sync.WaitGroup) {
			results <- process(path, pathType)
			<-goroutines
			wg.Done()
		}(path, *pathType, goroutines, &wg)
	}

	wg.Wait()
	close(goroutines)
	close(results)
	<-done
}
