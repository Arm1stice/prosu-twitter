package main

import (
	"os"

	redistore "gopkg.in/boj/redistore.v1"
)

func setupSessionStore() *redistore.RediStore {
	store, err := redistore.NewRediStore(10, "tcp", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PASSWORD"), []byte(os.Getenv("SESSION_SECRET")))
	if err != nil {
		panic(err)
	}
	return store
}
