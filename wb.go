package main

import (
	"flag"
	"net/http"
	"time"
	"fmt"
)

var (
	concurrent = flag.Int("c", 1, "Number of multiple requests to make")
	requests   = flag.Int("n", 1, "Number of requests to perform")
	verbosity  = flag.Bool("v", false, "Show info while running")
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

type Statistics struct {
	TimeMinimum	time.Duration
	TimeMaximum	time.Duration
	NumCalls	int
	TimeAggregate	int64

	StatusCodes	map[int]int
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

func (s *Statistics) Collector (idx int, rsp *URLResponse) {
	nsMin := s.TimeMinimum.Nanoseconds()
	nsMax := s.TimeMaximum.Nanoseconds()
	ns1 := rsp.Time.Nanoseconds()
	s.TimeAggregate = s.TimeAggregate + ns1	
	s.NumCalls++
	s.StatusCodes[rsp.Status] ++

	switch {
		case s.NumCalls == 1:
			s.TimeMinimum = rsp.Time
			s.TimeMaximum = rsp.Time
		case ns1 < nsMin:
			s.TimeMinimum = rsp.Time
		case ns1 > nsMax:
			s.TimeMaximum = rsp.Time
	}
	if *verbosity {
		mi := s.TimeMinimum / time.Millisecond
		mx := s.TimeMaximum / time.Millisecond
		av := (s.TimeAggregate / int64(s.NumCalls)) / time.Millisecond.Nanoseconds()
		call := rsp.Time / time.Millisecond
		fmt.Printf ("%6d: Call: %6dms|Min: %6dms|Avg: %6dms|Max: %6dms\n",(idx+1),call, mi,av,mx)
	}
}

func (s *Statistics) Dump () {
	mi := s.TimeMinimum / time.Millisecond
	mx := s.TimeMaximum / time.Millisecond
	av := (s.TimeAggregate / int64(s.NumCalls)) / time.Millisecond.Nanoseconds()
	fmt.Printf ("Min: %6dms\tAvg: %6dms\tMax: %6dms\n",mi,av,mx)
}

func NewStatistics () *Statistics {
	s := new (Statistics)
	s.StatusCodes = make(map[int]int)
	return s
}

func responseDumper (rsp *URLResponse) {
	//log.Printf ("%#v\n", rsp)
}

func main() {
	flag.Parse()
	stats := NewStatistics()
	var requestList = make (chan *URLRequest, *requests)
	var responseList = make (chan *URLResponse, *requests)

	for c:=0; c<*concurrent; c++ {
		makeFetcher (requestList, responseList)
	}
	u := flag.Arg(0)
	for i:=0; i<*requests; i++ {
		r := URLRequest {
			URL:u}
		requestList <- &r
      	}
	for i:=0; i<*requests; i++ {
		rsp := <-responseList
		stats.Collector (i,rsp)
		responseDumper (rsp)
	}
	stats.Dump ()
}
