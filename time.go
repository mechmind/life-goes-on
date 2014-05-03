package main

import (
	"time"
)

type Ticker interface {
	Tick(tick int64)
}

type Time struct {
	ticker Ticker
	freq   int64
	clock  *time.Ticker
}

func NewTime(freq int64, ticker Ticker) *Time {
	return &Time{ticker, freq, time.NewTicker(time.Second / time.Duration(freq))}
}

func (t *Time) Run() {
	var counter int64
	for _ = range t.clock.C {
		t.ticker.Tick(counter)
		counter++
	}
}
