package main

import (
	"net/http"
	"os"

	"github.com/newrelic/go-agent"
)

var config newrelic.Config
var nrApp newrelic.Application
var nrEnabled = false

func init() {
	if os.Getenv("ENVIRONMENT") == "production" {
		if os.Getenv("NEWRELIC_KEY") == "" {
			if os.Getenv("NEWRELIC_DISABLED") != "true" {
				panic("NEWRELIC_KEY must be set if deploying in production or NEWRELIC_DISABLED must be set to 'true'")
				return
			} else {
				log.Debug("NEWRELIC_KEY isn't set but NEWRELIC_DISABLED is set to true, ignoring NewRelic setup")
			}
		} else {
			config = newrelic.NewConfig("Prosu for Twitter", os.Getenv("NEWRELIC_KEY"))
			app, err := newrelic.NewApplication(config)
			if err != nil {
				panic(err)
			} else {
				nrEnabled = true
				nrApp = app
				log.Debug("Successfully registered NewRelic application")
			}
		}
	} else {
		log.Debug("Prosu is not running in a production environment, not enabling NewRelic")
	}
}

func relicHandle(location string, handler func(http.ResponseWriter, *http.Request)) (string, func(http.ResponseWriter, *http.Request)) {
	if nrEnabled {
		return newrelic.WrapHandleFunc(nrApp, location, handler)
	}
	return location, handler
}
