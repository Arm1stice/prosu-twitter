package main

import (
	"net/http"

	"github.com/go-chi/chi/middleware"
)

// When someone visits the home page
func homePage(w http.ResponseWriter, r *http.Request) {
	session, sessionError := sessionStore.Get(r, "prosu_session")
	if sessionError != nil {
		log.Error("There was an error getting the user's session")
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Error getting user session\nRequestID: "+reqID, http.StatusInternalServerError)
		return
	}

	session.Save(r, w)

	templates.ExecuteTemplate(w, "index.html", basicInterface{session})
}
