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

func fmin(v1, v2 float32) float32 {
	if v1 < v2 {
		return v1
	}
	return v2
}

func fmax(v1, v2 float32) float32 {
	if v1 > v2 {
		return v1
	}
	return v2
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

func iabs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func sgn(value float32) int {
	if value < 0 {
		return -1
	}
	return 1
}

func logPanic() {
	if err := recover(); err != nil {
		log.Println("main: recovering err:", err)
		var stack = make([]byte, 4096)
		n := runtime.Stack(stack, false)
		log.Println(string(stack[:n]))
	}
}
