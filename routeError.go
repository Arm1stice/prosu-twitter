package main

import (
	"net/http"
	"os"

	"github.com/getsentry/raven-go"
)

type errorPageData struct {
	Error     string
	RequestID string
	Code      int
}

var environment string

func init() {
	environment = os.Getenv("ENVIRONMENT")
	if environment == "production" {
		ravenKey := os.Getenv("RAVEN_URL")
		raven.SetDSN(ravenKey)
	}
}
func routeError(w http.ResponseWriter, e string, err error, rID string, code int) {
	data := errorPageData{
		Error:     e,
		RequestID: rID,
		Code:      code,
	}
	if err.Error() != "User Error" {
		if environment == "production" {
			raven.CaptureError(err, nil)
		}
	}

	w.WriteHeader(code)
	templates.ExecuteTemplate(w, "Error.html", data)
}

func captureError(err error) {
	if environment == "production" {
		raven.CaptureError(err, nil)
	}
}
