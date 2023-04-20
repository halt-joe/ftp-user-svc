package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSystemPost(t *testing.T) {
	params := map[string]string{"system": "BillSys1"}

	PostBody := "{\"id\": \"123\", \"ftp_id\": 987}"

	type args struct {
		w              *httptest.ResponseRecorder
		r              *http.Request
		expectedStatus int
		expectedBody   string
	}
	tests := []struct {
		name string
		args func(t *testing.T) args
	}{
		{
			name: "[ch408] Test return on mappings POST",
			args: func(t *testing.T) args {
				return args{
					w:              httptest.NewRecorder(),
					r:              httptest.NewRequest("POST", "https://ftpsvc.dev.run/mappings/BillSys1", strings.NewReader(PostBody)),
					expectedStatus: 201,
					expectedBody:   "{\"system\":\"BillSys1\",\"id\":\"123\",\"ftp_account\":{\"id\":987,\"username\":\"Test\",\"description\":\"A test user\"}}",
				}
			},
		},
	}

	env := Env{Data: &mockDB{}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			env.systemPostWithVars(tArgs.w, tArgs.r, params)
			resp := tArgs.w.Result()
			if resp.StatusCode != tArgs.expectedStatus {
				t.Errorf("Expected status %d but received %d", tArgs.expectedStatus, resp.StatusCode)
			}
			respBody, _ := io.ReadAll(resp.Body)
			body := string(respBody)
			if tArgs.expectedBody != body {
				t.Errorf("Expected body of %s but received %s", tArgs.expectedBody, body)
			}
		})
	}
}

func (mdb *mockDB) SystemIDUserRetrieve(system string) (map[string]string, error) {
	var result map[string]string

	if system == "BillSys1" {
		result = map[string]string{"system_id1": "username1", "system_id2": "username2"}
	}

	return result, nil
}

func TestSystemGet(t *testing.T) {
	type args struct {
		w              *httptest.ResponseRecorder
		r              *http.Request
		params         map[string]string
		expectedStatus int
		expectedBody   string
	}
	tests := []struct {
		name string
		args func(t *testing.T) args
	}{
		{
			name: "[ch3193] Test mappings GET Success",
			args: func(t *testing.T) args {
				system := "BillSys1"
				return args{
					w:              httptest.NewRecorder(),
					r:              httptest.NewRequest("GET", "https://ftpsvc.dev.run/mappings/"+system, nil),
					params:         map[string]string{"system": system},
					expectedStatus: http.StatusOK,
					expectedBody:   "{\"system_id1\":\"username1\",\"system_id2\":\"username2\"}",
				}
			},
		},
		{
			name: "[ch3193] Test mappings GET Bad Request",
			args: func(t *testing.T) args {
				system := ""
				return args{
					w:              httptest.NewRecorder(),
					r:              httptest.NewRequest("GET", "https://ftpsvc.dev.run/mappings/"+system, nil),
					params:         map[string]string{"system": system},
					expectedStatus: http.StatusBadRequest,
					expectedBody:   "{\"status\":400,\"location\":\"handlers.(*Env).systemGetWithVars\",\"message\":\"" + ErrSystemRequired + "\",\"error\":\"\"}",
				}
			},
		},
		{
			name: "[ch3193] Test mappings GET Not Found",
			args: func(t *testing.T) args {
				system := "CV3"
				return args{
					w:              httptest.NewRecorder(),
					r:              httptest.NewRequest("GET", "https://ftpsvc.dev.run/mappings/"+system, nil),
					params:         map[string]string{"system": system},
					expectedStatus: http.StatusNotFound,
					expectedBody:   "{\"status\":404,\"location\":\"handlers.(*Env).systemGetWithVars\",\"message\":\"" + fmt.Sprintf(ErrSystemNotFound, system) + "\",\"error\":\"\"}",
				}
			},
		},
	}

	env := Env{Data: &mockDB{}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			env.systemGetWithVars(tArgs.w, tArgs.r, tArgs.params)
			resp := tArgs.w.Result()
			if resp.StatusCode != tArgs.expectedStatus {
				t.Errorf("Expected status %d but received %d", tArgs.expectedStatus, resp.StatusCode)
			}
			respBody, _ := io.ReadAll(resp.Body)
			body := string(respBody)
			if tArgs.expectedBody != body {
				t.Errorf("Expected body of %s but received %s", tArgs.expectedBody, body)
			}
		})
	}
}
