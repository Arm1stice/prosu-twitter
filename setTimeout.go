package main

import "time"

// Code taken from https://www.loxodrome.io/post/set-timeout-interval-go/
func setTimeout(someFunc func(), milliseconds int) {

	timeout := time.Duration(milliseconds) * time.Millisecond

	// This spawns a goroutine and therefore does not block
	time.AfterFunc(timeout, someFunc)

}
