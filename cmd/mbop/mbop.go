package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redhatinsights/mbop/internal/config"
	"github.com/redhatinsights/mbop/internal/service/mailer"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"

	"github.com/go-chi/chi/v5"
	"github.com/redhatinsights/mbop/internal/handlers"
	l "github.com/redhatinsights/mbop/internal/logger"
	"github.com/redhatinsights/mbop/internal/middleware"
	"github.com/redhatinsights/mbop/internal/store"
)

var conf = config.Get()

func main() {
	if err := l.Init(); err != nil {
		panic(err)
	}

	if err := store.SetupStore(); err != nil {
		panic(err)
	}

	r := chi.NewRouter()
	// Emulating the log message at the beginning of mainHandler()
	r.Use(middleware.Logging)

	// TODO: move these to actual handler functions as we figure out which paths
	// are get vs post
	r.Get("/", handlers.Status)
	r.Get("/v*", handlers.CatchAll)
	r.Post("/v*", handlers.CatchAll)
	r.Get("/api/entitlements*", handlers.CatchAll)
	r.Get("/v1/jwt", handlers.JWTV1Handler)
	r.Post("/v1/users", handlers.UsersV1Handler)
	r.Post("/v1/sendEmails", handlers.SendEmails)
	r.Get("/v3/accounts/{orgID}/users", handlers.AccountsV3UsersHandler)
	r.Post("/v3/accounts/{orgID}/usersBy", handlers.AccountsV3UsersByHandler)
	r.Get("/v1/auth", handlers.AuthV1Handler)

	// all the handlers that need xrhid
	r.With(identity.EnforceIdentity).Group(func(r chi.Router) {
		r.Get("/v1/registrations", handlers.RegistrationListHandler)
		r.Post("/v1/registrations", handlers.RegistrationCreateHandler)
		r.Delete("/v1/registrations/{uid}", handlers.RegistrationDeleteHandler)
		r.Get("/v1/registrations/token", handlers.TokenHandler)

		r.Get("/api/mbop/v1/allowlist", handlers.AllowlistListHandler)
		r.Post("/api/mbop/v1/allowlist", handlers.AllowlistCreateHandler)
		r.Delete("/api/mbop/v1/allowlist", handlers.AllowlistDeleteHandler)
	})

	err := mailer.InitConfig()
	if err != nil {
		// TODO: should we panic if the mailer module fails to init?
		l.Log.Info("failed to init mailer module", "error", err)
	}

	// listen for OS signals so we can terminate when receiving one
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGTERM)

	go func() {
		srv := http.Server{
			Addr:              ":" + conf.Port,
			ReadHeaderTimeout: 2 * time.Second,
			Handler:           r,
		}

		l.Log.Info("Starting MBOP HTTP Listener", "port", conf.Port)
		if err := srv.ListenAndServe(); err != nil {
			l.Log.Error(err, "server couldn't start")
		}
	}()

	if conf.UseTLS {
		go func() {
			srv := http.Server{
				Addr:              ":" + conf.TLSPort,
				ReadHeaderTimeout: 2 * time.Second,
				Handler:           r,
			}

			l.Log.Info("Starting MBOP HTTPS Listener", "port", conf.TLSPort)
			if err := srv.ListenAndServeTLS(conf.CertDir+"/tls.crt", conf.CertDir+"/tls.key"); err != nil {
				l.Log.Error(err, "server couldn't start")
			}
		}()
	}

	<-interrupts
}
