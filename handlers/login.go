package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	sftpgo "github.com/drakkan/sftpgo/v2/dataprovider"
	"github.com/halt-joe/ftp-user-svc/apierror"
	"github.com/halt-joe/ftp-user-svc/auth"
	"github.com/halt-joe/ftp-user-svc/data"
	"github.com/halt-joe/ftp-user-svc/metrics"
)

// GetUserNameFromLoginRequest - read in the body of a login request and return the username
func GetUserNameFromLoginRequest(r *http.Request) string {
	result := "Unknown"

	// Read Body
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		return result
	}

	// Unmarshall
	var creds data.Credentials
	err = json.Unmarshal(b, &creds)
	if err != nil {
		return result
	}

	// create a new ReadCloser for the handler
	r.Body = io.NopCloser(bytes.NewBuffer(b))

	result = creds.Username

	return result
}

// LoginHandler - validates the provided credentials against the FTP User entries
//
//	 Responses:
//		  - 200 Success
//		  - 401 Unauthorized (Failed Authentication)
//		  - 500 Internal Server Error
//
//	 Request Body:
//	   {username":"testuser", "password":"testpassword"}
//
//	 Response Body:
//	   {id:234, "status":1, "username":"testuser", "password":"testPassword", "description":"Test Description"}
func (env *Env) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// setup error response
	er := apierror.NewErrorResponse(w, r)

	// Authenticate
	if !auth.Authenticate(r) {
		metrics.IncLoginTotals(metrics.LoginStatusAuthFailure)
		er.Status = http.StatusUnauthorized
		er.Message = auth.ErrUnauthorized
		er.WriteResponse()
		return
	}

	// Read Body
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		metrics.IncLoginTotals(metrics.LoginStatusServerError)
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	// Unmarshall
	var creds data.Credentials
	err = json.Unmarshal(b, &creds)
	if err != nil {
		metrics.IncLoginTotals(metrics.LoginStatusServerError)
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	// Empty Username or Password not valid
	if creds.Username == "" || creds.Password == "" {
		metrics.IncLoginTotals(metrics.LoginStatusUserPassBlank)
		er.User = creds.Username
		er.Status = http.StatusUnauthorized
		er.Message = auth.ErrUnauthorized
		er.WriteResponse()
		return
	}

	// Look for User in Database
	user, err := env.Data.FtpUserLookup(creds.Username)
	if err != nil {
		e := err.Error()
		if e == data.ErrUserNotFound {
			metrics.IncLoginTotals(metrics.LoginStatusUserNotFound)
			er.User = creds.Username
			er.Status = http.StatusUnauthorized
			er.Message = auth.ErrUnauthorized
			er.WriteResponse()
			return
		}
		metrics.IncLoginTotals(metrics.LoginStatusServerError)
		er.User = creds.Username
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	if user.Password != creds.Password {
		metrics.IncLoginTotals(metrics.LoginStatusBadPassword)
		er.User = creds.Username
		er.Status = http.StatusUnauthorized
		er.Message = auth.ErrUnauthorized
		er.WriteResponse()
		return
	}

	metrics.IncLoginTotals(metrics.LoginStatusSuccess)

	user.Status = 1

	// set user permissions to list and download only
	user.Permissions = map[string][]string{"/": {sftpgo.PermListItems, sftpgo.PermDownload}}

	output, err := json.Marshal(user)
	if err != nil {
		er.User = user.Username
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}
