package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/halt-joe/ftp-user-svc/apierror"
	"github.com/halt-joe/ftp-user-svc/auth"
	"github.com/halt-joe/ftp-user-svc/data"
)

// Custom Errors
const (
	ErrFTPMappingRequired = "System, SystemID and FTP_ID are all required"
	ErrSystemRequired     = "System is required"
	ErrSystemNotFound     = "System %s not found"
)

// SystemIDDelete - removes a mapping related to the provided system and id
//
//	Responses:
//	  - 204 successfully deleted
//	  - 401 Unauthorized (Failed Authentication)
//	  - 404 system id not found
//	  - 500 Error
//
//	Request Path Parameters:
//	  /mappings/{system}/{id}
//	- system
//	    the system that the mapping is associated with e.g. "BillSys1
//	- id
//	    the id in the {system} mapped to the ftp user
func (env *Env) SystemIDDelete(w http.ResponseWriter, r *http.Request) {
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
	system := params["system"]
	id := params["id"]

	rows, err := env.Data.MappingDelete(system, id)

	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	if rows == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SystemIDGet - retrieves mapping related to the provided system and id
//
//	Responses:
//	  - 200 Success
//	  - 401 Unauthorized (Failed Authentication)
//	  - 404 system id not found
//	  - 500 Error
//
//	Request Path Parameters:
//	  /mappings/{system}/{id}
//	- system
//	    the system that the mapping is associated with e.g. "BillSys1
//	- id
//	    the id in the {system} mapped to the ftp user
func (env *Env) SystemIDGet(w http.ResponseWriter, r *http.Request) {
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
	system := params["system"]
	id := params["id"]

	mapping, err := env.Data.MappingRetrieve(system, id)
	if err != nil {
		e := err.Error()
		if e == data.ErrMappingNotFound {
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

	output, err := json.Marshal(mapping)
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

// SystemPost - creates or updates mapping related to the provided system
//
//	            for the provided system_id and ftp_id
//
//	Responses:
//	  - 200 Updated
//	  - 201 Created
//	  - 400 Bad Request
//	  - 401 Unauthorized (Failed Authentication)
//	  - 404 The requested ftp_id does not exist
//	  - 500 Error
//
//	Request:
//	  /mappings/{system}
//	- system
//	    the system that the mapping is associated with e.g. "BillSys1
//
//	Request Body:
//	  {"id": 999, "ftp_id": 7}
//	- id
//	    the id in the {system} mapped to the ftp user
//	- ftp_id
//	    the id of the ftp account to map
func (env *Env) SystemPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	env.systemPostWithVars(w, r, vars)
}

func (env *Env) systemPostWithVars(w http.ResponseWriter, r *http.Request, params map[string]string) {
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
	var mapping data.NewMapping
	err = json.Unmarshal(b, &mapping)
	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	mapping.System = params["system"]

	if mapping.System == "" || mapping.SystemID == "" || mapping.FTPAccountID == 0 {
		fmt.Printf("params: %s", params)
		er.Status = http.StatusBadRequest
		er.Message = ErrFTPMappingRequired
		er.WriteResponse()
		return
	}

	result, err := env.Data.MappingCreate(mapping)
	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	switch result {
	case data.MappingFTPAccountNotFound:
		w.WriteHeader(http.StatusNotFound)
		return

	case data.MappingInserted:
		user, err := env.Data.FtpUserGet(mapping.FTPAccountID)
		if err != nil {
			er.Status = http.StatusInternalServerError
			er.Err = err
			er.WriteResponse()
			return
		}

		var result data.Mapping
		result.ID = mapping.SystemID
		result.System = mapping.System
		result.FTPAccount.ID = user.ID
		result.FTPAccount.Username = user.Username
		result.FTPAccount.Description = user.Description
		result.FTPAccount.Password = ""

		output, err := json.Marshal(result)
		if err != nil {
			er.Status = http.StatusInternalServerError
			er.Err = err
			er.WriteResponse()
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(output)
		return

	case data.MappingUpdated:
		w.WriteHeader(http.StatusOK)
		return
	}
}

// SystemGet - retieves the system_id and username pairs related to the provided system
//
//	Responses:
//	  - 200 OK
//	  - 400 Bad Request
//	  - 401 Unauthorized (Failed Authentication)
//	  - 404 The provided system does not exist
//	  - 500 Error
//
//	Request:
//	  /mappings/{system}
//	- system
//	    the system that the system_id and username pairs are associated with e.g. "BillSys1
func (env *Env) SystemGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	env.systemGetWithVars(w, r, vars)
}

func (env *Env) systemGetWithVars(w http.ResponseWriter, r *http.Request, params map[string]string) {
	// setup error response
	er := apierror.NewErrorResponse(w, r)

	// Authenticate
	if !auth.Authenticate(r) {
		er.Status = http.StatusUnauthorized
		er.Message = auth.ErrUnauthorized
		er.WriteResponse()
		return
	}

	system := params["system"]

	if system == "" {
		fmt.Printf("params: %s", params)
		er.Status = http.StatusBadRequest
		er.Message = ErrSystemRequired
		er.WriteResponse()
		return
	}

	result, err := env.Data.SystemIDUserRetrieve(system)
	if err != nil {
		er.Status = http.StatusInternalServerError
		er.Err = err
		er.WriteResponse()
		return
	}

	if len(result) < 1 {
		er.Status = http.StatusNotFound
		er.Message = fmt.Sprintf(ErrSystemNotFound, system)
		er.WriteResponse()
		return
	}

	output, err := json.Marshal(result)
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
