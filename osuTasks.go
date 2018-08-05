package main

import (
	"context"

	osuapi "github.com/wcalandro/osuapi-go"
	"golang.org/x/time/rate"
)

type osuRateLimiter struct {
	API osuapi.API
}

var limit rate.Limit
var rLimiter *rate.Limiter

func init() {
	limit = rate.Limit(250 / 60) // 250 API Calls every 60 seconds
	rLimiter = rate.NewLimiter(limit, 10)
}
func newOsuLimiter(api osuapi.API) osuRateLimiter {
	return osuRateLimiter{
		API: api,
	}
}

func (limiter osuRateLimiter) GetUser(options osuapi.M) (*osuapi.User, error) {
	rLimiter.Wait(context.Background())
	return limiter.API.GetUser(options)
}
