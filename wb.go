package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var (
	concurrent = flag.Int("c", 1, "Number of multiple requests to make")
	requests   = flag.Int("n", 1, "Number of requests to perform")
	verbosity  = flag.Int("v", 0, "Show info while running")
)

const (
	VERBOSE_NOTHING = 0
	VERBOSE_MIN     = 1
	VERBOSE_MAX     = 2
)

type URLRequest struct {
	URL string
}

type URLResponse struct {
	Time          time.Duration
	Status        int
	ContentLength int64
	Body          string
	Error         error
}

type URLFetcher struct {
	Client *http.Client
}

type Statistics struct {
	TimeMinimum   time.Duration
	TimeMaximum   time.Duration
	NumCalls      int
	TimeAggregate int64

	StatusCodes map[int]int
}

func (c *URLFetcher) loadURL(rq *URLRequest) (*URLResponse, error) {
	response := new(URLResponse)
	t0 := time.Now()
	rsp, err := c.Client.Get(rq.URL)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()
	body, err := ioutil.ReadAll(rsp.Body)
	response.Time = time.Since(t0)
	if err != nil {
		return nil, err
	}
	response.Body = string(body)
	response.Status = rsp.StatusCode
	response.ContentLength = rsp.ContentLength
	if rsp.ContentLength == -1 {
		response.ContentLength = int64(len(body))
	}
	return response, nil
}

func makeFetcher(f *URLFetcher, inc chan *URLRequest, out chan *URLResponse) {
	go func() {
		for rq := range inc {
			rsp, err := f.loadURL(rq)
			if err != nil {
				rsp = new(URLResponse)
				rsp.Error = err
			}
			out <- rsp
		}
	}()
}

func (s *Statistics) Collector(idx int, rsp *URLResponse) {
	nsMin := s.TimeMinimum.Nanoseconds()
	nsMax := s.TimeMaximum.Nanoseconds()
	ns1 := rsp.Time.Nanoseconds()
	s.TimeAggregate = s.TimeAggregate + ns1
	s.NumCalls++
	s.StatusCodes[rsp.Status]++

	switch {
	case s.NumCalls == 1:
		s.TimeMinimum = rsp.Time
		s.TimeMaximum = rsp.Time
	case ns1 < nsMin:
		s.TimeMinimum = rsp.Time
	case ns1 > nsMax:
		s.TimeMaximum = rsp.Time
	}
	if *verbosity == VERBOSE_MAX {
		if rsp.Error != nil {
			fmt.Printf("%#v\n", rsp.Error)
		}
		mi := s.TimeMinimum / time.Millisecond
		mx := s.TimeMaximum / time.Millisecond
		av := (s.TimeAggregate / int64(s.NumCalls)) / time.Millisecond.Nanoseconds()
		call := rsp.Time / time.Millisecond
		fmt.Printf("%6d: Call: %6dms|Min: %6dms|Avg: %6dms|Max: %6dms|Stat:%3d|Len:%6d\n", (idx + 1), call, mi, av, mx, rsp.Status, rsp.ContentLength)
	}
	if *verbosity == VERBOSE_MIN {
		if s.NumCalls%100 == 0 {
			s.Dump()
		}
	}
}

func (s *Statistics) Dump() {
	mi := s.TimeMinimum / time.Millisecond
	mx := s.TimeMaximum / time.Millisecond
	av := (s.TimeAggregate / int64(s.NumCalls)) / time.Millisecond.Nanoseconds()
	fmt.Printf("Calls: %6d\tMin: %6dms\tAvg: %6dms\tMax: %6dms\n", s.NumCalls, mi, av, mx)
}

func NewStatistics() *Statistics {
	s := new(Statistics)
	s.StatusCodes = make(map[int]int)
	return s
}

func responseDumper(rsp *URLResponse) {
	//log.Printf ("%#v\n", rsp)
}

func NewFetcher() *URLFetcher {
	fetcher := new(URLFetcher)
	tr := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	fetcher.Client = &http.Client{Transport: tr}
	return fetcher
}

func main() {
	flag.Parse()
	stats := NewStatistics()
	requestList := make(chan *URLRequest, *requests)
	responseList := make(chan *URLResponse, *requests)
	fetcher := NewFetcher()

	for c := 0; c < *concurrent; c++ {
		makeFetcher(fetcher, requestList, responseList)
	}
	u := flag.Arg(0)
	for i := 0; i < *requests; i++ {
		r := URLRequest{
			URL: u}
		requestList <- &r
	}
	for i := 0; i < *requests; i++ {
		rsp := <-responseList
		stats.Collector(i, rsp)
		responseDumper(rsp)
	}
	fmt.Println("Result:")
	fmt.Printf("URL: %s\n", u)
	stats.Dump()
}
