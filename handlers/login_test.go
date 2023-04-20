package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginPost(t *testing.T) {
	// rqstBody := "{\"id\": \"123\", \"ftp_id\": 987}"
	rqstBody := "{\"username\": \"Test\", \"password\": \"pass\"}"

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
			name: "Test bad username on login POST",
			args: func(t *testing.T) args {
				return args{
					w:              httptest.NewRecorder(),
					r:              httptest.NewRequest("POST", "https://ftpsvc.dev.run/login/", strings.NewReader("{\"username\": \"bad-name\", \"password\": \"bad-pass\"}")),
					expectedStatus: 401,
					expectedBody:   "{\"status\":401,\"location\":\"handlers.(*Env).LoginHandler\",\"message\":\"Unauthorized (Failed Authentication)\",\"error\":\"\"}",
				}
			},
		},
		{
			name: "Test success on login POST",
			args: func(t *testing.T) args {
				return args{
					w:              httptest.NewRecorder(),
					r:              httptest.NewRequest("POST", "https://ftpsvc.dev.run/login/", strings.NewReader(rqstBody)),
					expectedStatus: 200,
					expectedBody:   "{\"id\":987,\"status\":1,\"username\":\"Test\",\"expiration_date\":0,\"password\":\"pass\",\"home_dir\":\"\",\"uid\":0,\"gid\":0,\"max_sessions\":0,\"quota_size\":0,\"quota_files\":0,\"permissions\":{\"/\":[\"list\",\"download\"]},\"created_at\":0,\"updated_at\":0,\"description\":\"A test user\",\"filters\":{\"hooks\":{\"external_auth_disabled\":false,\"pre_login_disabled\":false,\"check_password_disabled\":false},\"totp_config\":{}},\"filesystem\":{\"provider\":0,\"s3config\":{},\"gcsconfig\":{},\"azblobconfig\":{},\"cryptconfig\":{},\"sftpconfig\":{}}}",
				}
			},
		},
	}

	env := Env{Data: &mockDB{}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			env.LoginHandler(tArgs.w, tArgs.r)
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
