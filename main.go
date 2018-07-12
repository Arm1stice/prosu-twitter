package main

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/go-bongo/bongo"

	"github.com/gorilla/sessions"
	"github.com/mrjones/oauth"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/context"
	"github.com/joho/godotenv"
	"github.com/op/go-logging"
	redistore "gopkg.in/boj/redistore.v1"
)

var log = logging.MustGetLogger("prosu")

/* Set up session store */
var sessionStore *redistore.RediStore

/* Set up templates */
var templates = template.Must(template.ParseGlob("templates/*"))

/* Basic interface */
type basicInterface struct {
	session *sessions.Session
}

/* Twitter Consumer Key and Secret */
var consumerKey string
var consumerSecret string
var twitterConsumer *oauth.Consumer

// Domain
var domain string

type sessionTokenStorer struct {
	Token *oauth.RequestToken
}

// Database variables
var mongoConnectionString = os.Getenv("MONGODB_CONNECTION_STRING")
var mongoDatabase = os.Getenv("MONGO_DATABASE")
var connection *bongo.Connection

func init() {
	/* First, setting up logging */
	loggingBackend := logging.NewLogBackend(os.Stdout, "", 0)
	format := logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
	loggingBackendFormatter := logging.NewBackendFormatter(loggingBackend, format)
	logging.SetBackend(loggingBackendFormatter)
	/* Second, as long as we aren't in the production environment, try to load a .env for configuration */
	if os.Getenv("ENVIRONMENT") != "production" {
		if err := godotenv.Load(); err != nil {
			log.Warning("Couldn't load .env file")
			fmt.Println(err)
		} else {
			log.Info("Successfully loaded .env file")
		}
	} else {
		log.Debug("Production environment, not loading .env")
	}

	// Initialize sessionStore
	sessionStore = setupSessionStore()

	// Initialize consumer key and secret
	consumerKey = os.Getenv("CONSUMER_KEY")
	consumerSecret = os.Getenv("CONSUMER_SECRET")

	// Initialize domain
	domain = os.Getenv("DOMAIN")

	// Initialize Twitter client
	twitterConsumer = oauth.NewConsumer(
		consumerKey,
		consumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "https://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authenticate",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		},
	)

	// Add sessionTokenStorer to gob
	gob.Register(sessionTokenStorer{})

	// Connect to MongoDB
	conn, err := bongo.Connect(&bongo.Config{
		ConnectionString: mongoConnectionString,
		Database:         mongoDatabase,
	})
	if err != nil {
		panic(err)
	}

	connection = conn
}

func main() {
	/* Set up chi router */
	// Initialize the router
	r := chi.NewRouter()

	// Initialize the middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", homePage)
	r.Get("/connect/twitter", redirectToTwitter)
	r.Get("/connect/twitter/callback", obtainAccessToken)

	//FileServer(r, "/assets", http.Dir("./static"))

	r.Get("/favicon.ico", ServeFavicon)
	/* Listen */
	port := "5000"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}
	http.ListenAndServe(":"+port, context.ClearHandler(r))
}
