package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/halt-joe/ftp-user-svc/apierror"
	"github.com/halt-joe/ftp-user-svc/auth"
	"github.com/halt-joe/ftp-user-svc/data"
)

// Custom Errors
const (
	ErrInvalidFTPUserID    = "Invalid FTP User ID"
	ErrFTPUserRequired     = "Username, Description and Password are all required"
	ErrFTPUserUpdateReq    = "Username and Description are both required"
	ErrFTPUserPasswordReq  = "Password is required"
	ErrFTPUserIDConversion = "Cannot convert %s to an integer"
	ErrFTPAccountExists    = "An FTP Account for %s already exists"
	ErrFTPUserNotFound     = "User Not Found"
)

// Get - retrieves all ftp user accounts within a specified page index and page size
//
//	Responses:
//	  - 200 Success
//	  - 401 Unauthorized (Failed Authentication)
//	  - 500 Error
//
//	Response Body:
//	  {
//	    "ftpusers": [
//	      {"id":11,"username":"testuser","description":"test description"},
//	      {"id":12,"username":"testuser2","description":"test description 2"},
//	      ...
//	    ],
//	    "total_items": 245,
//	    "total_pages": 13
//	  }
func (env *Env) Get(w http.ResponseWriter, r *http.Request) {
	var (
		page     uint32
		pageSize uint32
		search   string
	)

	// setup error response
	er := apierror.NewErrorResponse(w, r)

	// Authenticate
	if !auth.Authenticate(r) {
		er.Status = http.StatusUnauthorized
		er.Message = auth.ErrUnauthorized
		er.WriteResponse()
		return
	}

	// read in the query params
	if value := r.FormValue("page"); value != "" {
		i, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			er.Status = http.StatusInternalServerError
			er.Err = err
			er.WriteResponse()
			return
		}
		page = uint32(i)
	}
	if value := r.FormValue("page_size"); value != "" {
		i, err := strconv.ParseUint(value, 10, 32)
		if err != nil {
			er.Status = http.StatusInternalServerError
			er.Err = err
			er.WriteResponse()
			return
		}
		pageSize = uint32(i)
	}
	if value := r.FormValue("q"); value != "" {
		search = value
	}

	users, err := env.Data.FtpUserGetSelection(page, pageSize, search)

	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	output, err := json.Marshal(users)
	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}

// IDGet - retrieve the ftp user account associated with the provided id
//
//	Responses:
//	  - 200 Success
//	  - 400 Bad Request
//	  - 401 Unauthorized (Failed Authentication)
//	  - 404 User Not Found
//	  - 500 Error
//
//	Request Path Parameters:
//	  /ftpusers/{id}
//	- id
//	    the id of the ftp user entry
//
//	Response Body:
//	  {"id":11,"username":"testuser","description":"test description"}
func (env *Env) IDGet(w http.ResponseWriter, r *http.Request) {
	// setup error response
	er := apierror.NewErrorResponse(w, r)

	// Authenticate
	if !auth.Authenticate(r) {
		er.Status = http.StatusUnauthorized
		er.Message = auth.ErrUnauthorized
		er.WriteResponse()
		return
	}

	params := mux.Vars(r)
	id, err := strconv.ParseInt(params["id"], 10, 32)

	if err != nil {
		msg := fmt.Sprintf(ErrFTPUserIDConversion, params["id"])
		er.Status = http.StatusBadRequest
		er.Message = msg
		er.Err = err
		er.WriteResponse()
		return
	}

	if id < 1 {
		er.Status = http.StatusBadRequest
		er.Message = ErrInvalidFTPUserID
		er.WriteResponse()
		return
	}
	user, err := env.Data.FtpUserGet(uint32(id))
	if err != nil {
		e := err.Error()
		if e == data.ErrUserNotFound {
			er.Status = http.StatusNotFound
			er.Message = e
			er.WriteResponse()
			return
		}
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	output, err := json.Marshal(user)
	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}

// Post - create an FTP User
//
//	Responses:
//	  - 201 Created
//	  - 400 Bad Request
//	  - 401 Unauthorized (Failed Authentication)
//	  - 409 Conflict
//	  - 500 Error
//
//	Request Body:
//	  {username":"testuser", "description":"test description", "password":"testpassword"}
//
//	Response Body:
//	  {"id":11,"username":"testuser","description":"test description"}
func (env *Env) Post(w http.ResponseWriter, r *http.Request) {
	// setup error response
	er := apierror.NewErrorResponse(w, r)

	// Authenticate
	if !auth.Authenticate(r) {
		er.Status = http.StatusUnauthorized
		er.Message = auth.ErrUnauthorized
		er.WriteResponse()
		return
	}

	// Read Body
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	// Unmarshall
	var user data.FtpUser
	err = json.Unmarshal(b, &user)
	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	// Empty Username, Description or Password not valid
	if user.Username == "" || user.Description == "" || user.Password == "" {
		er.User = user.Username
		er.Status = http.StatusBadRequest
		er.Message = ErrFTPUserRequired
		er.WriteResponse()
		return
	}

	id, err := env.Data.FtpUserCreate(user)
	if err != nil {
		e := err.Error()
		if e == data.ErrFTPAccountExists {
			er.User = user.Username
			er.Status = http.StatusConflict
			er.Message = fmt.Sprintf(ErrFTPAccountExists, user.Username)
			er.Err = err
			er.WriteResponse()
			return
		}
		er.User = user.Username
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	user.ID = id
	user.Password = ""

	output, err := json.Marshal(user)
	if err != nil {
		er.User = user.Username
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(output)
}

// IDPut - update an FTP User specified by id
//
//	Responses:
//	  - 200 Success
//	  - 400 Bad Request
//	  - 401 Unauthorized (Failed Authentication)
//	  - 404 Not Found
//	  - 500 Error
//
//	Request Path Parameters:
//	  /ftpusers/{id}
//	- id
//	    the id of the ftp user entry
//
//	Request Body:
//	  {username":"testuser", "description":"test description"}
func (env *Env) IDPut(w http.ResponseWriter, r *http.Request) {
	// setup error response
	er := apierror.NewErrorResponse(w, r)

	// Authenticate
	if !auth.Authenticate(r) {
		er.Status = http.StatusUnauthorized
		er.Message = auth.ErrUnauthorized
		er.WriteResponse()
		return
	}

	params := mux.Vars(r)
	id, err := strconv.ParseInt(params["id"], 10, 32)

	if err != nil {
		er.Status = http.StatusBadRequest
		er.Message = fmt.Sprintf(ErrFTPUserIDConversion, params["id"])
		er.Err = err
		er.WriteResponse()
		return
	}

	if id < 1 {
		er.Status = http.StatusBadRequest
		er.Message = ErrInvalidFTPUserID
		er.WriteResponse()
		return
	}

	// Read Body
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	// Unmarshall
	var user data.FtpUser
	err = json.Unmarshal(b, &user)
	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	// Empty Username or Description is not valid
	if user.Username == "" || user.Description == "" {
		er.Status = http.StatusBadRequest
		er.Message = ErrFTPUserUpdateReq
		er.WriteResponse()
		return
	}

	user.ID = uint32(id)

	err = env.Data.FtpUserUpdate(user)
	if err != nil {
		e := err.Error()
		if e == data.ErrFTPAccountNotFound {
			er.Status = http.StatusNotFound
			er.Message = e
			er.WriteResponse()
			return
		}
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	user.Password = ""

	output, err := json.Marshal(user)
	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(output)
}

// IDDelete - delete an FTP User specified by id
//
//	Responses:
//	  - 204 No Content
//	  - 400 Bad Request
//	  - 401 Unauthorized (Failed Authentication)
//	  - 404 Not Found
//	  - 500 Error
//
//	Request Path Parameters:
//	  /ftpusers/{id}
//	- id
//	    the id of the ftp user entry
func (env *Env) IDDelete(w http.ResponseWriter, r *http.Request) {
	// setup error response
	er := apierror.NewErrorResponse(w, r)

	// Authenticate
	if !auth.Authenticate(r) {
		er.Status = http.StatusUnauthorized
		er.Message = auth.ErrUnauthorized
		er.WriteResponse()
		return
	}

	params := mux.Vars(r)
	id, err := strconv.ParseInt(params["id"], 10, 32)

	if err != nil {
		er.Status = http.StatusBadRequest
		er.Message = fmt.Sprintf(ErrFTPUserIDConversion, params["id"])
		er.Err = err
		er.WriteResponse()
		return
	}

	if id < 1 {
		er.Status = http.StatusBadRequest
		er.Message = ErrInvalidFTPUserID
		er.WriteResponse()
		return
	}

	err = env.Data.FtpUserDelete(uint32(id))

	if err != nil {
		e := err.Error()
		if e == data.ErrFTPAccountNotFound {
			er.Status = http.StatusNotFound
			er.Message = e
			er.WriteResponse()
			return
		}
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// IDPatch - update the password of the FTP User specified by id
//
//	Responses:
//	  - 200 Success
//	  - 400 Bad Request
//	  - 401 Unauthorized (Failed Authentication)
//	  - 404 Not Found
//	  - 500 Error
//
//	Request Path Parameters:
//	  /ftpusers/{id}
//	- id
//	    the id of the ftp user entry
//
//	Request Body:
//	  {"password":"newpassword"}
func (env *Env) IDPatch(w http.ResponseWriter, r *http.Request) {
	// setup error response
	er := apierror.NewErrorResponse(w, r)

	// Authenticate
	if !auth.Authenticate(r) {
		er.Status = http.StatusUnauthorized
		er.Message = auth.ErrUnauthorized
		er.WriteResponse()
		return
	}

	params := mux.Vars(r)
	id, err := strconv.ParseInt(params["id"], 10, 32)

	if err != nil {
		er.Status = http.StatusBadRequest
		er.Message = fmt.Sprintf(ErrFTPUserIDConversion, params["id"])
		er.Err = err
		er.WriteResponse()
		return
	}

	if id < 1 {
		er.Status = http.StatusBadRequest
		er.Message = ErrInvalidFTPUserID
		er.WriteResponse()
		return
	}

	// Read Body
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	// Unmarshall
	var user data.FtpUser
	err = json.Unmarshal(b, &user)
	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	// Empty Password is not valid
	if user.Password == "" {
		er.Status = http.StatusBadRequest
		er.Message = ErrFTPUserPasswordReq
		er.WriteResponse()
		return
	}

	user.ID = uint32(id)

	err = env.Data.FtpUserUpdatePassword(user)
	if err != nil {
		e := err.Error()
		if e == data.ErrFTPAccountNotFound {
			er.Status = http.StatusNotFound
			er.Message = e
			er.WriteResponse()
			return
		}
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	w.WriteHeader(http.StatusOK)
}
