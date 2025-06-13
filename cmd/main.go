package main

import (
	"expvar"
	"runtime"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/georgifotev1/bms/internal/db"
	"github.com/georgifotev1/bms/internal/env"
	"github.com/georgifotev1/bms/internal/mailer"
	"github.com/georgifotev1/bms/internal/ratelimiter"
	"github.com/georgifotev1/bms/internal/store"
	"github.com/georgifotev1/bms/internal/store/cache"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

const version = "1.1.0"

// @title						Event Managing System
// @description				API for Event Managing System
// @termsOfService				http://swagger.io/terms/
// @contact.name				API Support
// @contact.url				http://www.swagger.io/support
// @contact.email				support@swagger.io
// @license.name				Apache 2.0
// @license.url				http://www.apache.org/licenses/LICENSE-2.0.html
//
// @BasePath					/v1
//
// @securityDefinitions.apikey	CookieAuth
// @in							cookie
// @name						session_id
// @description				Session-based authentication using cookies
func main() {
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	err := godotenv.Load(".env")
	if err != nil {
		logger.Error(err)
	}

	cfg := config{
		address:    env.GetString("ADDR", ":8080"),
		apiUrl:     env.GetString("EXTERNAL_URL", "localhost:8080"),
		clientUrl:  env.GetString("FRONTEND_URL", "localhost:5173"),
		clientHost: env.GetString("FRONTEND_HOST", "localhost"),
		env:        env.GetString("ENV", "development"),
		db: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost/bms?sslmode=disable"),
			maxOpenConns: 30,
			maxIdleConns: 30,
			maxIdleTime:  "15m",
		},
		auth: authConfig{
			basic: basicConfig{
				user: env.GetString("AUTH_BASIC_USER", "admin"),
				pass: env.GetString("AUTH_BASIC_PASS", "admin"),
			},
			session: sessionConfig{
				exp: time.Hour,
			},
		},
		mail: mailConfig{
			exp:       time.Hour * 24,
			fromEmail: env.GetString("FROM_EMAIL", ""),
			mailTrap: mailTrapConfig{
				apiKey: env.GetString("MAILTRAP_API_KEY", ""),
			},
		},
		cache: redisConfig{
			addr:    env.GetString("REDIS_ADDR", "localhost:6379"),
			pw:      env.GetString("REDIS_PW", ""),
			db:      env.GetInt("REDIS_DB", 0),
			enabled: env.GetBool("REDIS_ENABLED", false),
		},
		rateLimiter: ratelimiter.Config{
			RequestsPerTimeFrame: env.GetInt("RATELIMITER_REQUESTS_COUNT", 10),
			TimeFrame:            time.Second * 5,
			Enabled:              env.GetBool("RATE_LIMITER_ENABLED", true),
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

	var rdb *redis.Client
	if cfg.cache.enabled {
		rdb = cache.NewRedisClient(cfg.cache.addr, cfg.cache.pw, cfg.cache.db)
		logger.Info("redis cache connection established")

		defer rdb.Close()
	}

	store := store.NewStore(db)
	redisCache := cache.NewRedisStorage(rdb)

	mailtrap, err := mailer.NewMailTrapClient(cfg.mail.mailTrap.apiKey, cfg.mail.fromEmail)
	if err != nil {
		logger.Fatal(err)
	}

	rateLimiter := ratelimiter.NewFixedWindowLimiter(
		cfg.rateLimiter.RequestsPerTimeFrame,
		cfg.rateLimiter.TimeFrame,
	)

	cld, err := cloudinary.New()
	if err != nil {
		logger.Fatal(err)
	}

	app := &application{
		config:       cfg,
		store:        store,
		logger:       logger,
		mailer:       mailtrap,
		cache:        redisCache,
		rateLimiter:  rateLimiter,
		imageService: cld,
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
