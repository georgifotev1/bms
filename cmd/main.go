package main

import (
	"expvar"
	"runtime"
	"time"

	"github.com/georgifotev1/bms/internal/db"
	"github.com/georgifotev1/bms/internal/env"
	"github.com/georgifotev1/bms/internal/store"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

const version = "1.1.0"

//	@title			BMS
//	@version		1.0.0
//	@description	Booking Management System API
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.example.com/support
//	@contact.email	support@example.com

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8080
//	@BasePath	/v1

// @securityDefinitions.apikey	ApiKeyAuth
// @in							cookie
// @name						session_cookie
func main() {
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	err := godotenv.Load(".env")
	if err != nil {
		logger.Error(err)
	}

	cfg := config{
		address:   env.GetString("ADDR", ":8080"),
		apiUrl:    env.GetString("EXTERNAL_URL", "localhost:8080"),
		clientUrl: env.GetString("FRONTEND_URL", "http://localhost:3000"),
		env:       env.GetString("ENV", "development"),
		db: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost/bms?sslmode=disable"),
			maxOpenConns: 30,
			maxIdleConns: 30,
			maxIdleTime:  "15m",
		},
		auth: authConfig{
			basic: basicCfg{
				user: env.GetString("AUTH_BASIC_USER", "admin"),
				pass: env.GetString("AUTH_BASIC_PASS", "admin"),
			},
			token: tokenCfg{
				secret: env.GetString("AUTH_TOKEN_SECRET", "supersecret"),
				exp:    time.Hour * 24,
				iss:    "bms",
			},
		},
	}

	db, err := db.New(
		cfg.db.addr,
		cfg.db.maxOpenConns,
		cfg.db.maxIdleConns,
		cfg.db.maxIdleTime,
	)
	if err != nil {
		logger.Fatal(err)
	}

	defer db.Close()
	logger.Info("database connection pool established")

	store := store.New(db)

	app := &application{
		config: cfg,
		store:  *store,
		logger: logger,
	}

	expvar.NewString("version").Set(version)
	expvar.Publish("database", expvar.Func(func() any {
		return db.Stats()
	}))
	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	mux := app.mount()

	logger.Fatal(app.run(mux))
}
