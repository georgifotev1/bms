package main

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/georgifotev1/bms/docs"
	"github.com/georgifotev1/bms/internal/auth"
	"github.com/georgifotev1/bms/internal/mailer"
	"github.com/georgifotev1/bms/internal/ratelimiter"
	"github.com/georgifotev1/bms/internal/store"
	"github.com/georgifotev1/bms/internal/store/cache"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"go.uber.org/zap"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type application struct {
	config      config
	store       store.Store
	cache       cache.Storage
	logger      *zap.SugaredLogger
	mailer      mailer.Client
	auth        auth.Authenticator
	rateLimiter ratelimiter.Limiter
	cloudinary  *cloudinary.Cloudinary
}

type config struct {
	address     string
	db          dbConfig
	auth        authConfig
	mail        mailConfig
	env         string
	apiUrl      string
	clientUrl   string
	clientHost  string
	cache       redisConfig
	rateLimiter ratelimiter.Config
}

type dbConfig struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type redisConfig struct {
	addr    string
	pw      string
	db      int
	enabled bool
}

type authConfig struct {
	basic basicConfig
	token tokenConfig
}

type tokenConfig struct {
	secret string
	exp    time.Duration
	iss    string
}

type basicConfig struct {
	user string
	pass string
}

type mailConfig struct {
	mailTrap  mailTrapConfig
	fromEmail string
	exp       time.Duration
}

type mailTrapConfig struct {
	apiKey string
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{fmt.Sprintf("http://app.%v", app.config.clientUrl), fmt.Sprintf("http://*.%v", app.config.clientUrl)},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "User-Agent"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	if app.config.rateLimiter.Enabled {
		r.Use(app.RateLimiterMiddleware)
	}

	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", app.healthCheckHandler)
		r.With(app.BasicAuthMiddleware()).Get("/debug/vars", expvar.Handler().ServeHTTP)

		docsURL := fmt.Sprintf("%s/swagger/doc.json", app.config.address)
		r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL(docsURL)))

		r.Route("/users", func(r chi.Router) {
			r.Get("/confirm/{token}", app.activateUserHandler)

			r.Route("/", func(r chi.Router) {
				r.Use(app.AuthTokenMiddleware)
				r.Get("/", app.getUsersHandler)
				r.Get("/me", app.getUserProfile)
				r.Post("/invite", app.inviteUserHandler)
				r.Get("/{id}", app.getUserHandler)
			})
		})

		r.Route("/brand", func(r chi.Router) {
			r.With(app.AuthTokenMiddleware).Post("/", app.createBrandHandler)
			r.Get("/{id}", app.getBrandHandler)
		})

		r.Route("/service", func(r chi.Router) {
			r.With(app.AuthTokenMiddleware).Post("/", app.createServiceHandler)
			r.With(app.AuthTokenMiddleware).Put("/{serviceId}", app.updateServiceHandler)
			r.Route("/{brandId}", func(r chi.Router) {
				r.Get("/", app.getServicesHandler)
			})
		})

		r.Route("/events", func(r chi.Router) {
			r.Post("/", app.createEventHandler)
			r.With(app.AuthTokenMiddleware).Get("/timestamp", app.getEventsByTimeStampHandler)
			r.With(app.AuthTokenMiddleware).Put("/{eventId}", app.updateEventHandler)
		})

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", app.registerUserHandler)
			r.Post("/token", app.createTokenHandler)
			r.Get("/refresh", app.refreshTokenHandler)
			r.Post("/logout", app.logoutHandler)
		})

		r.Route("/customers", func(r chi.Router) {
			r.With(app.AuthTokenMiddleware).Get("/", app.getCustomersHandler)
			r.Post("/guest", app.createGuestCustomerHandler)
			r.Route("/auth", func(r chi.Router) {
				r.Use(app.BrandMiddleware)
				r.Post("/register", app.registerCustomerHandler)
				r.Post("/login", app.loginCustomerHandler)
				r.Get("/refresh", app.refreshCustomerTokenHandler)
				r.Post("/logout", app.logoutCustomerHandler)
			})
		})
	})

	return r
}

func (app *application) run(mux http.Handler) error {
	docs.SwaggerInfo.Version = version
	docs.SwaggerInfo.Host = app.config.apiUrl
	docs.SwaggerInfo.BasePath = "/v1"

	srv := &http.Server{
		Addr:         app.config.address,
		Handler:      mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	shutdown := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		app.logger.Infow("signal caught", "signal", s.String())
		shutdown <- srv.Shutdown(ctx)
	}()

	app.logger.Infow("server has started", "addr", app.config.address, "env", app.config.env)

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdown
	if err != nil {
		return err
	}

	app.logger.Infow("server has stopped", "addr", app.config.address, "env", app.config.env)

	return nil
}
