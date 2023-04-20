package apierror

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/halt-joe/ftp-user-svc/metrics"
	log "github.com/inconshreveable/log15"
)

// ContextKey is used for a context.Context value. The value requires a key that is not a primitive type.
type ContextKey string

// ContextKeyRequestID is the ContextKey for RequestID
const ContextKeyRequestID ContextKey = "requestID"

// apiError - struct used to create json response
type apiError struct {
	Status   int    `json:"status"`
	Location string `json:"location"`
	Message  string `json:"message"`
	Error    string `json:"error"`
}

// ErrorResponse - struct used to encapsulate necessary data for error handling
type ErrorResponse struct {
	Writer    http.ResponseWriter
	RequestID string
	User      string
	Status    int
	Message   string
	Err       error
}

// NewErrorResponse - create a new error response struct with the provided writer and requestId
func NewErrorResponse(writer http.ResponseWriter, r *http.Request) ErrorResponse {
	requestID := ""
	ok := false

	ctx := r.Context()

	reqID := ctx.Value(ContextKeyRequestID)

	if requestID, ok = reqID.(string); ok == false {
		requestID = ""
	}

	return ErrorResponse{
		Writer:    writer,
		RequestID: requestID,
	}
}

func formatLocation(location string) string {
	return strings.Replace(location, "github.com/halt-joe/ftp-user-svc/", "", 1)
}

func (er *ErrorResponse) logAPIError(ae apiError) {
	output := fmt.Sprintf("%s", er.RequestID)

	if er.User != "" {
		output += fmt.Sprintf(" User: %s", er.User)
	}

	location := formatLocation(ae.Location)

	output += fmt.Sprintf(" Status: %d Location: %s", ae.Status, location)

	if ae.Message != "" {
		output += fmt.Sprintf(" Message: %s", ae.Message)
	}

	if ae.Error != "" {
		output += fmt.Sprintf(" Error: %s", ae.Error)
	}

	log.Error(output)

	return
}

// WriteResponse - creates the apiError response and logs the error
func (er *ErrorResponse) WriteResponse() {
	var ae apiError

	metrics.IncError()

	// grab the calling function
	pc, _, _, _ := runtime.Caller(1)
	ae.Location = formatLocation(runtime.FuncForPC(pc).Name())

	ae.Status = er.Status
	ae.Message = er.Message
	if er.Err != nil {
		ae.Error = er.Err.Error()
	}

	output, err := json.Marshal(ae)
	if err != nil {
		e := err.Error()
		ae.Error += " [" + e + "]"
		er.logAPIError(ae)
		http.Error(er.Writer, e, http.StatusInternalServerError)
		return
	}

	er.logAPIError(ae)

	er.Writer.Header().Set("Content-Type", "application/json")
	er.Writer.WriteHeader(er.Status)
	er.Writer.Write(output)

	return
}
