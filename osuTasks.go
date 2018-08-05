package main

import (
	osuapi "github.com/wcalandro/osuapi-go"
	"golang.org/x/time/rate"
)

type osuRateLimiter struct {
	API         osuapi.API
	RateLimiter *rate.Limiter
}

const limit float64 = 250 / 60 // 250 API Calls every 60 seconds

func newOsuLimiter(api osuapi.API) osuRateLimiter {
	return osuRateLimiter{
		API:         api,
		RateLimiter: rate.NewLimiter(rate.Limit(limit), 10),
	}
}

func (limiter osuRateLimiter) GetUser(options osuapi.M) (*osuapi.User, error) {
	limiter.RateLimiter.Wait(nil)
	return limiter.API.GetUser(options)
}
