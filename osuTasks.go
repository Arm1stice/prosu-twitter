package main

import (
	"context"

	osuapi "github.com/wcalandro/osuapi-go"
	"golang.org/x/time/rate"
)

type osuRateLimiter struct {
	API     osuapi.API
	Limiter *rate.Limiter
}

func newOsuLimiter(api osuapi.API, callsPerMinute int) osuRateLimiter {
	rLimiter := rate.NewLimiter(rate.Limit(callsPerMinute/60), 10)
	return osuRateLimiter{
		API:     api,
		Limiter: rLimiter,
	}
}

func (limiter osuRateLimiter) GetUser(options osuapi.M) (*osuapi.User, error) {
	limiter.Limiter.Wait(context.Background())
	return limiter.API.GetUser(options)
}
