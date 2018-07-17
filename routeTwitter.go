package main

import (
	"net/http"
	"net/url"
	"os"

	"github.com/ChimeraCoder/anaconda"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/sessions"
	"github.com/mrjones/oauth"
)

// Redirect a user to Twitter to authenticate if they haven't authenticated already
func redirectToTwitter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Error getting user session\nRequestID: "+reqID, http.StatusInternalServerError)
		return
	}
	session := ctx.Value("session").(*sessions.Session)
	isAuthenticated := ctx.Value("isAuthenticated").(bool)

	// See if the user is already authenticated
	if isAuthenticated {
		// User is already authenticated, redirect them home
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	protocol := "https://"
	if os.Getenv("ENVIRONMENT") != "production" {
		protocol = "http://"
	}

	token, url, err := twitterConsumer.GetRequestTokenAndUrl(protocol + domain + "/connect/twitter/callback")
	if err != nil {
		log.Error("There was an error generating the URL to redirect the user to for Twitter authorization")
		log.Error(err.Error())
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Error generating Twitter redirect URL\nRequestID: "+reqID, http.StatusInternalServerError)
		return
	}

	session.Values["twitter_token"] = sessionTokenStorer{token}
	err = session.Save(r, w)
	if err != nil {
		log.Error("There was an error saving the session")
		log.Error(err.Error())
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Error saving session\nRequestID: "+reqID, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// Obtain user's access token when Twitter redirects us
func obtainAccessToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessionError := ctx.Value("session_error").(string)
	if sessionError != "" {
		log.Error("There was an error getting the user's session")
		log.Error(sessionError)
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Error getting user session\nRequestID: "+reqID, http.StatusInternalServerError)
		return
	}
	session := ctx.Value("session").(*sessions.Session)

	twitterRequestToken := (session.Values["twitter_token"].(sessionTokenStorer)).Token

	// Check if the user has a token stored in their session
	if twitterRequestToken == nil {
		log.Error("User doesn't have a twitter token located in their session")
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "No token detected in your session\nRequestID: "+reqID, http.StatusBadRequest)
		return
	}

	query := r.URL.Query()
	verificationCode := query.Get("oauth_verifier")
	twitterTokenKey := query.Get("oauth_token")

	if verificationCode == "" {
		log.Error("No oauth_verifier returned in callback")
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "No oauth_verifier returned in callback\nRequestID: "+reqID, http.StatusBadRequest)
		return
	}

	if twitterTokenKey == "" {
		log.Error("No oauth_token returned in callback")
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "No oauth_token returned in callback\nRequestID: "+reqID, http.StatusBadRequest)
		return
	}

	if twitterTokenKey != twitterRequestToken.Token {
		log.Error("Twitter oauth_token mismatch")
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Twitter oauth_token mismatch\nRequestID: "+reqID, http.StatusBadRequest)
		return
	}

	// Finally, obtain the access token
	twitterAccessToken, err := twitterConsumer.AuthorizeToken(twitterRequestToken, verificationCode)
	if err != nil {
		log.Error("Error obtaining access token")
		log.Error(err.Error())
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Error obtaining access token\nRequestID: "+reqID, http.StatusInternalServerError)
		return
	}

	twitterUser, err := getUserInfo(twitterAccessToken)
	if err != nil {
		log.Error("Error getting Twitter user info")
		log.Error(err.Error())
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Error getting Twitter user info\nRequestID: "+reqID, http.StatusInternalServerError)
		return
	}

	user, err := findOrCreateUser(twitterUser, twitterAccessToken)
	if err != nil {
		log.Error("Error finding or creating user")
		log.Error(err.Error())
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Error finding or creating user\nRequestID: "+reqID, http.StatusInternalServerError)
		return
	}
	session.Values["isAuthenticated"] = true
	session.Values["user_id"] = user.GetId().Hex()
	if err := session.Save(r, w); err != nil {
		log.Error("Error saving session after the user was logged in")
		log.Error(err.Error())
		ctx := r.Context()
		reqID := middleware.GetReqID(ctx)
		http.Error(w, "Error saving session after being logged in\nRequestID: "+reqID, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func getUserInfo(accessToken *oauth.AccessToken) (anaconda.User, error) {
	api := anaconda.NewTwitterApiWithCredentials(accessToken.Token, accessToken.Secret, consumerKey, consumerSecret)
	values := url.Values{}
	values.Set("skip_status", "true")
	return api.GetSelf(values)
}
