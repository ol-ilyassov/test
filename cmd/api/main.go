package main

import (
	"context"
	"database/sql"
	"expvar"
	"flag"
	"github.com/ol-ilyassov/test/internal/data"
	"github.com/ol-ilyassov/test/internal/jsonlog"
	"html/template"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Application Version number
const version = "1.0.0"

// Configuration Settings
type config struct {
	port int    // Network Port
	env  string // Current Operating Environment
	db   struct {
		dsn          string // Database Connection
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	limiter struct {
		rps     float64 // Request per second
		burst   int     // Number of maximum request in single burst
		enabled bool    // Is RateLimiter turned On
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
}

// Dependencies for HTTP handlers, helpers, and middleware
type application struct {
	config        config
	logger        *jsonlog.Logger
	models        data.Models
	wg            sync.WaitGroup
	templateCache map[string]*template.Template
}

func main() {
	// Instance of config struct
	var cfg config

	// Default values
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	// Env variable
	// .bashrc - file in the root folder (as in hierarchy of linux)
	//flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")
	//flag.StringVar(&cfg.db.dsn, "db-dsn", "postgres://postgres:123@localhost/greenlight?sslmode=disable", "PostgreSQL DSN")

	//flag.StringVar(&cfg.db.dsn, "db-dsn", "root:@/daryn?parseTime=true", "MySQL DSN") //&sslmode=disable

	//flag.StringVar(&cfg.db.dsn, "db-dsn", "admin_newdaryn:3D8Bc5yG1K@tcp(89.218.185.158:3306)/admin_newdaryn?parseTime=true", "MySQL DSN")
	// 89.218.185.158:3306
	//admin_newdaryn

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "8db0d4c74dccb8", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "8376f58c61e62a", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "RIG <no-reply@rig.mail.net>", "SMTP sender")

	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	flag.Parse()

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Create Connection Pool
	//db, err := openDB(cfg)
	//if err != nil {
	//	logger.PrintFatal(err, nil)
	//}
	//defer db.Close()
	//logger.PrintInfo("database connection pool established", nil)

	// Npw: cmdline, memstats, version.
	expvar.NewString("version").Set(version)

	// Num of Active goroutines
	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))

	// DB pool Statistics
	//expvar.Publish("database", expvar.Func(func() interface{} {
	//	return db.Stats()
	//}))

	// Current Unix timestamp
	expvar.Publish("timestamp", expvar.Func(func() interface{} {
		return time.Now().Unix()
	}))

	// Template Cache init
	templateCache, err := newTemplateCache("./ui/html/")
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	// Instance of application struct
	app := &application{
		config:        cfg,
		logger:        logger,
		models:        data.NewModels(),
		templateCache: templateCache,
	}

	err = app.serve()
	if err != nil {
		logger.PrintFatal(err, nil)
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	//if err = db.Ping(); err != nil {
	//	return nil, err
	//}
	return db, nil
}
