package main

import (
	"go-sniffer/core"
	"go-sniffer/pkg/queue"
)

func main() {
	c := core.New()
	go queue.Metrics.Export()
	c.Run()
}
