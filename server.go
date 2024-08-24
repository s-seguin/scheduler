package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"scheduler/auth"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/sessions"
)

type Server struct {
	Router        *chi.Mux
	Store         *sessions.CookieStore
	Authenticator *auth.Authenticator
}

func NewServer() (*Server, error) {
	router := chi.NewRouter()
	store := sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

	authenticator, err := auth.New()
	if err != nil {
		return nil, err
	}

	server := &Server{
		Router:        router,
		Store:         store,
		Authenticator: authenticator,
	}

	return server, nil
}

func (s *Server) SetupMiddleware() {
	s.Router.Use(middleware.Logger)
	s.Router.Use(s.addUserToContext)
}

func (s *Server) MountRouter(path string, router *chi.Mux) {
	s.Router.Mount(path, router)
}

type Auth0Profile struct {
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Name          string    `json:"name"`
	FamilyName    string    `json:"family_name"`
	GivenName     string    `json:"given_name"`
	Nickname      string    `json:"nickname"`
	Picture       string    `json:"picture"`
	UpdatedAt     time.Time `json:"updated_at"`
	Sub           string    `json:"sub"`
	Expiry        int64     `json:"exp"`
	Iat           int64     `json:"iat"`
	Locale        string    `json:"locale"`
	Iss           string    `json:"iss"`
}

func (ap Auth0Profile) IsExpired() bool {
	exp := time.Unix(ap.Expiry, 0)
	return time.Now().After(exp)
}

func (s *Server) MountAuthRoutes() {
	gob.Register(Auth0Profile{})

	s.Router.Get("/login", func(w http.ResponseWriter, r *http.Request) {
		// todo should we use crypto/rand
		randBytes := make([]byte, 32)
		_, err := rand.Read(randBytes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		state := base64.StdEncoding.EncodeToString(randBytes)

		session, _ := s.Store.Get(r, "auth-session")
		session.Values["state"] = state
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		authURL := s.Authenticator.AuthCodeURL(state)

		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	})

	s.Router.Get("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		session, _ := s.Store.Get(r, "auth-session")
		state := session.Values["state"]
		if state != r.URL.Query().Get("state") {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		oauth2Token, err := s.Authenticator.Exchange(r.Context(), r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		idToken, err := s.Authenticator.VerifyIDToken(r.Context(), oauth2Token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var profile Auth0Profile
		err = idToken.Claims(&profile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		session.Values["access_token"] = oauth2Token.AccessToken
		session.Values["profile"] = profile
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("profile: %+v\n", profile)

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})

	s.Router.Get("/logout", func(w http.ResponseWriter, r *http.Request) {
		logoutUrl, err := url.Parse("https://" + os.Getenv("AUTH0_DOMAIN") + "/v2/logout")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}

		returnTo, err := url.Parse(scheme + "://" + r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		params := url.Values{}
		params.Add("returnTo", returnTo.String())
		params.Add("client_id", os.Getenv("AUTH0_CLIENT_ID"))
		logoutUrl.RawQuery = params.Encode()

		// destroy the session
		session, _ := s.Store.Get(r, "auth-session")
		session.Options.MaxAge = -1
		err = session.Save(r, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})
}

func (s *Server) ServeStatic(path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		s.Router.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	s.Router.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

// fixme -- define this somewhere better
type RequestContextKey string

func (s *Server) addUserToContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, _ := s.Store.Get(r, "auth-session")
		profile := session.Values["profile"]

		// todo -- we should probably check the expiry here
		if profile != nil {
			ctx := context.WithValue(r.Context(), RequestContextKey("profile"), profile)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) Start() {
	log.Println("Starting server on :8080")
	http.ListenAndServe(":8080", s.Router)
}
