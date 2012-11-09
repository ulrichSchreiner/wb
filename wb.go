package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"
	"log"
)

var (
	concurrent = flag.Int("c", 1, "Number of multiple requests to make")
	requests   = flag.Int("n", 1, "Number of requests to perform")
)

type URLRequest struct {
	URL string
}

type URLResponse struct {
	Time          time.Duration
	Status        int
	ContentLength int64
	Error	error
}

type URLFetcher struct {
	Client http.Client
}

func (c *URLFetcher) loadURL(rq *URLRequest) (*URLResponse, error) {
	response := new(URLResponse)
	t0 := time.Now()
	rsp, err := c.Client.Get(rq.URL)
	response.Time = time.Since(t0)
	if err != nil {
		return nil, err
	}
	response.Status = rsp.StatusCode
	response.ContentLength = rsp.ContentLength

	return response, nil
}

func makeFetcher (inc chan *URLRequest, out chan *URLResponse) {
	f := new (URLFetcher)
	go func () {
		for rq := range(inc) {
			rsp, err := f.loadURL(rq)
			if err != nil {
				rsp = new (URLResponse)
				rsp.Error = err
			}
			out <- rsp
		}
	} ()	
}

func responseDumper (rsp *URLResponse) {
	log.Printf ("%#v\n", rsp)
}

var requestList = make (chan *URLRequest, 100)
var responseList = make (chan *URLResponse, 100)

func main() {
	makeFetcher (requestList, responseList)
	r := URLRequest {
		URL:"http://www.google.de/"}
	requestList <- &r
	for rsp := range (responseList) {
		responseDumper (rsp)
	}
	fmt.Println("Hallo")
}
