package main

import "net/http"

type errorPageData struct {
	Error     string
	RequestID string
	Code      int
}

func routeError(w http.ResponseWriter, e string, rID string, code int) {
	data := errorPageData{
		Error:     e,
		RequestID: rID,
		Code:      code,
	}
	w.WriteHeader(code)
	templates.ExecuteTemplate(w, "Error.html", data)
}
