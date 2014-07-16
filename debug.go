package main

import (
	"net/http"
	_ "net/http/pprof"
)

func runDebugAt(addr string) {
	http.ListenAndServe(addr, nil)
}
