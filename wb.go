package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

var (
	concurrent = flag.Int("c", 1, "Number of concurrent requests to make")
	requests   = flag.Int("n", 1, "Number of requests to perform")
	verbosity  = flag.Int("v", 0, "Show info while running")
	reuse      = flag.Bool("r",false, "Reuse HTTP Client")
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
	StatusCodes   map[int]int
	Errors        []error
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
	if rsp.Status == 0 {
		// Call was illegal, so don't count
		s.Errors = append(s.Errors, rsp.Error)
		return
	}
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
		fmt.Printf("%6d: Call: %6dms|Min: %6dms|Avg: %6dms|Max: %6dms|Stat:%3d|Len:%6d|%s\n", (idx + 1), call, mi, av, mx, rsp.Status, rsp.ContentLength,rsp.Body)
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
	var av int64
	if s.NumCalls > 0 {
		av = (s.TimeAggregate / int64(s.NumCalls)) / time.Millisecond.Nanoseconds()
	}
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
	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "Usage of %s <url>:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}
	u := flag.Arg(0)

	stats := NewStatistics()
	requestList := make(chan *URLRequest, *requests)
	responseList := make(chan *URLResponse, *requests)
	fetcherSingle := NewFetcher()
	for c := 0; c < *concurrent; c++ {
		fetcher := fetcherSingle
		if !*reuse {
			fetcher = NewFetcher()
		}
		makeFetcher(fetcher, requestList, responseList)
	}

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
	for k, v := range stats.StatusCodes {
		fmt.Printf("Status %3d: %d Calls\n", k, v)
	}
	for i, e := range stats.Errors {
		switch e.(type) {
		case *url.Error:
			ue := e.(*url.Error)
			if ne, ok := ue.Err.(net.Error); ok {
				switch {
				case ne.Temporary():
					fmt.Printf("Error %d is temporary\n", i)
				case ne.Timeout():
					fmt.Printf("Error %d is temporary\n", i)
				default:
					fmt.Printf("Error %d: %#v\n", i, ne)
				}
			} else {
				fmt.Printf("Error %d: %#v\n", i, ue.Err)
			}

		default:
			fmt.Printf("Error %d: %#v\n", i, e)
		}
	}
}
