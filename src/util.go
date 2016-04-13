package main

import (
	"net/http"
	"sync"
)

//Map is a key value pair
// it has strings for keys and int for values
var requestBytes map[string]int64
var requestLock sync.Mutex

func init() {
	requestBytes = make(map[string]int64)
}

// Update my stats (interally) it takes pointers, so hopefully we are only
// reading data from them (the request and the response
// We could use what is called a Channel in this case, but we are keeping it simple see :
// https://golang.org/doc/effective_go.html#channels
func UpdateStats(req *http.Request, resp *http.Response) int64 {
	requestLock.Lock()
	defer requestLock.Unlock()

	bytes := requestBytes[req.URL.Path] + resp.ContentLength
	requestBytes[req.URL.Path] = bytes
	return bytes
}
