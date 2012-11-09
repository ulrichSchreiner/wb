package main

import (
	"flag"
	"fmt"
)

var (
	concurrent = flag.Int("c", 1, "Number of multiple requests to make")
	requests   = flag.Int("n", 1, "Number of requests to perform")
)

func main() {
	fmt.Println("Hallo Welt")
}
