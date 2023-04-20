package main

import (
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/halt-joe/ftp-user-svc/auth"
	"github.com/halt-joe/ftp-user-svc/data"
	"github.com/halt-joe/ftp-user-svc/handlers"
	"github.com/halt-joe/ftp-user-svc/router"
	log "github.com/inconshreveable/log15"
	"github.com/rs/cors"
)

const xAPIKey = "myvalue"
const httpPort = "8080"

const dbConStr = "mysql://svcuser:svcpass@tcp(127.0.0.1:3308)/ftpusersvc"

// const dbConStr = "host=postgrestest port=5432 user=ftpsvc password=svcpass dbname=ftpusers sslmode=require"

// Azure Parameters
const (
	azKey       = "test-key=="
	azAccount   = "test-account"
	azContainer = "test-container"
)

// EnvVar - return value of envar or the defVal if not set
func EnvVar(envVar string, defVal string) string {
	value := os.Getenv(envVar)
	if value == "" {
		value = defVal
	}
	return value
}
func main() {
	err := sentry.Init(sentry.ClientOptions{})
	if err != nil {
		log.Error("Error Initializing sentry: ", "error", err.Error())
	}
	auth.APIKey = EnvVar("APIKEY", xAPIKey)

	log.Info("Server started")

	db, err := data.NewDB(EnvVar("DBCON", dbConStr))
	if err != nil {
		log.Crit(err.Error())
		sentry.CaptureException(err)
		sentry.Flush(time.Second * 5)
		return
	}
	defer db.Close()

	env := &handlers.Env{Data: db}

	data.AZKey = EnvVar("AZKEY", azKey)
	data.AZAccount = EnvVar("AZACCOUNT", azAccount)
	data.AZContainer = EnvVar("AZCONTAINER", azContainer)

	router := cors.AllowAll().Handler(router.Create(env))
	err = http.ListenAndServe(":"+EnvVar("HTTPPORT", httpPort), router)
	log.Crit(err.Error())
	sentry.CaptureException(err)
	sentry.Flush(time.Second * 5)
}
