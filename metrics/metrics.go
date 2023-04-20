package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Port specifies the port to be used by the listener
var Port string = ""

// Login Status Messages
const (
	LoginStatusSuccess       = "success"
	LoginStatusAuthFailure   = "authentication_failure"
	LoginStatusServerError   = "server_error"
	LoginStatusBadPassword   = "bad_password"
	LoginStatusUserPassBlank = "username_password_blank"
	LoginStatusUserNotFound  = "username_not_found"
)

var (
	loginLabels = prometheus.Labels{"status": ""}
	countErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ftpusersvc_errors_total",
			Help: "The total number of errors produced by the service",
		})
	countLoginTotals = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ftpusersvc_logins_total",
			Help: "The total number of login requests with a status to indicate success or an error message to indicate failure due to a problem with the supplied credentials"},
		[]string{"status"})
)

// IncError - increments the error counter by 1
func IncError() {
	countErrors.Inc()
}

// IncLoginTotals - increment the logins total counter with the provided label values
func IncLoginTotals(status string) {
	loginLabels["status"] = status
	countLoginTotals.With(loginLabels).Inc()
}
