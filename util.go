package main

import (
	"log"
	"runtime"
)

func fabs(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

func fbound(value, low, high float32) float32 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}

	return value
}

func ibound(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high-1 {
		return high - 1
	}

	return value
}

func logPanic() {
	if err := recover(); err != nil {
		log.Println("main: recovering err:", err)
		var stack = make([]byte, 4096)
		n := runtime.Stack(stack, false)
		log.Println(string(stack[:n]))
	}
}
