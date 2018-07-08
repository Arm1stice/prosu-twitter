package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/context"
	"github.com/joho/godotenv"
	"github.com/op/go-logging"
	redistore "gopkg.in/boj/redistore.v1"
)

var log = logging.MustGetLogger("prosu")
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

/* Set up session store */
var sessionStore *redistore.RediStore

func main() {
	/* First, setting up logging */
	loggingBackend := logging.NewLogBackend(os.Stderr, "", 0)
	loggingBackendFormatter := logging.NewBackendFormatter(loggingBackend, format)

	logging.SetBackend(loggingBackend, loggingBackendFormatter)

	/* Second, as long as we aren't in the production environment, try to load a .env for configuration */
	if os.Getenv("ENVIRONMENT") != "production" {
		if err := godotenv.Load(); err != nil {
			log.Warning("Couldn't load .env file")
			fmt.Println(err)
		} else {
			log.Info("Successfully loaded .env file")
		}
	}

	// Initialize sessionStore
	sessionStore = setupSessionStore()

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
	r.Get("/favicon.ico", serveFavicon)
	/* Listen */
	http.ListenAndServe(":8080", context.ClearHandler(r))
}

// When someone visits the home page
func homePage(w http.ResponseWriter, r *http.Request) {
	session, sessionError := sessionStore.Get(r, "prosu_session")
	if sessionError != nil {
		log.Error("There was an error getting the user's session")
	}
	session.Save(r, w)

	fmt.Fprintf(w, "Hello")
}

// Serve the favicon
func serveFavicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/favicon.ico")
}

func setupSessionStore() *redistore.RediStore {
	store, err := redistore.NewRediStore(10, "tcp", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PASSWORD"), []byte(os.Getenv("SESSION_SECRET")))
	if err != nil {
		panic(err)
	}
	return store
}
