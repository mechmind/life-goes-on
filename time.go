package main

import (
	"time"
)

const (
	TIME_TICKS_PER_SEC = 10
)

type Ticker interface {
	Tick(tick int64)
}

type Time struct {
	ticker Ticker
	freq   int64
	clock  *time.Ticker
	stopCh chan struct{}
}

func NewTime(freq int64) *Time {
	return &Time{nil, freq, time.NewTicker(time.Second / time.Duration(freq)),
		make(chan struct{})}
}

func (t *Time) Run() {
	var counter int64
	defer logPanic()
	for {
		select {
		case <-t.clock.C:
			t.ticker.Tick(counter)
			counter++
		case <-t.stopCh:
			return
		}
	}
}

func (t *Time) Stop() {
	t.stopCh <-struct{}{}
}

func (t *Time) SetTicker(tr Ticker) {
	t.ticker = tr
}
