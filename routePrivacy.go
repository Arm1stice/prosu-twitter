package main

import "net/http"

type emptyStruct struct {
}

func routePrivacy(w http.ResponseWriter, r *http.Request) {
	templates.ExecuteTemplate(w, "privacy.html", emptyStruct{})
}
