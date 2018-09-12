package main

import (
	"net/http"
	"os"

	rollbar "github.com/rollbar/rollbar-go"
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
		rollbar.SetToken(os.Getenv("ROLLBAR_TOKEN"))
		rollbar.SetEnvironment("production")
		rollbar.SetCodeVersion(os.Getenv("GIT_REV"))
		rollbar.SetServerRoot("github.com/wcalandro/prosu-twitter")
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
			rollbar.Critical(err)
			log.Critical(err.Error())
		} else {
			panic(err)
		}
	}

	w.WriteHeader(code)
	templates.ExecuteTemplate(w, "Error.html", data)
}

func captureError(err error) {
	if environment == "production" {
		rollbar.Critical(err)
		log.Critical(err.Error())
	} else {
		panic(err)
	}
}
