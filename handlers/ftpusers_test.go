package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	sftpgo "github.com/drakkan/sftpgo/v2/dataprovider"
	"github.com/halt-joe/ftp-user-svc/data"
)

const (
	errDBConnectionError = "an error '%s' was not expected when opening a stub database connection"
)

var errNotImplmented = errors.New("not implemented")

type mockDB struct{}

func (mdb *mockDB) FtpUserLookup(username string) (sftpgo.User, error) {
	if username == "Test" {
		user := sftpgo.User{}
		user.ID = 987
		user.Username = "Test"
		user.Description = "A test user"
		user.Password = "pass"
		return user, nil
	}
	return sftpgo.User{}, errors.New(data.ErrUserNotFound)
}
func (mdb *mockDB) MappingDelete(system string, id string) (int64, error) {
	return 0, nil
}
func (mdb *mockDB) MappingRetrieve(system string, id string) (data.Mapping, error) {
	return data.Mapping{}, errors.New("MappingRetrieve not implemented")
}
func (mdb *mockDB) MappingCreate(mapping data.NewMapping) (int, error) {
	return data.MappingInserted, nil
}
func (mdb *mockDB) FtpUserGetSelection(page uint32, pageSize uint32, search string) (data.FtpUsers, error) {
	return data.FtpUsers{}, errors.New("FtpUserGetSelection() Not implemented")
}
func (mdb *mockDB) FtpUserGet(id uint32) (data.FtpUser, error) {
	user, err := mdb.FtpUserLookup("Test")
	result := data.FtpUser{ID: uint32(user.ID), Username: user.Username, Description: user.Description, Password: user.Password}
	return result, err
}
func (mdb *mockDB) FtpUserCreate(user data.FtpUser) (uint32, error) {
	return 1, nil
}
func (mdb *mockDB) FtpUserUpdate(user data.FtpUser) error {
	return errNotImplmented
}
func (mdb *mockDB) FtpUserDelete(id uint32) error {
	return errNotImplmented
}
func (mdb *mockDB) FtpUserUpdatePassword(user data.FtpUser) error {
	return errNotImplmented
}
func TestGet(t *testing.T) {

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errDBConnectionError, err)
	}
	defer db.Close()

	type args struct {
		w              *httptest.ResponseRecorder
		r              *http.Request
		expectedStatus int
	}
	tests := []struct {
		name string
		args func(t *testing.T) args
	}{
		{
			name: "[ch428] Test return status on DB error",
			args: func(t *testing.T) args {
				return args{
					w:              httptest.NewRecorder(),
					r:              httptest.NewRequest("GET", "https://ftpsvc.dev.run/ftpusers", nil),
					expectedStatus: 500,
				}
			},
		},
	}

	env := Env{Data: &data.Database{DB: db}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			env.Get(tArgs.w, tArgs.r)
			resp := tArgs.w.Result()
			if resp.StatusCode != tArgs.expectedStatus {
				t.Errorf("Expected status %d but received %d", tArgs.expectedStatus, resp.StatusCode)
			}

		})
	}
}

func TestGetResponse(t *testing.T) {
	type params struct {
		url            string
		userCount      int
		expStatus      int
		expResultCount int
		expTotalItems  uint32
		expTotalPages  uint32
	}
	tests := []struct {
		name      string
		getParams func(t *testing.T) params
	}{
		{
			name: "[CH486] Test no parameters provided",
			getParams: func(t *testing.T) params {
				return params{
					url:            "https://localhost/ftpusers",
					userCount:      300,
					expStatus:      http.StatusOK,
					expResultCount: 30,
					expTotalItems:  300,
					expTotalPages:  10,
				}
			},
		},
		{
			name: "[CH486] Test only page_size parameter provided",
			getParams: func(t *testing.T) params {
				return params{
					url:            "https://localhost/ftpusers?page_size=5",
					userCount:      300,
					expStatus:      http.StatusOK,
					expResultCount: 5,
					expTotalItems:  300,
					expTotalPages:  60,
				}
			},
		},
		{
			name: "[CH486] Test only page parameter provided",
			getParams: func(t *testing.T) params {
				return params{
					url:            "https://localhost/ftpusers?page=7",
					userCount:      300,
					expStatus:      http.StatusOK,
					expResultCount: 30,
					expTotalItems:  300,
					expTotalPages:  10,
				}
			},
		},
		{
			name: "[CH486] Test smallest page_size",
			getParams: func(t *testing.T) params {
				return params{
					url:            "https://localhost/ftpusers?page=4&page_size=1",
					userCount:      300,
					expStatus:      http.StatusOK,
					expResultCount: 1,
					expTotalItems:  300,
					expTotalPages:  300,
				}
			},
		},
		{
			name: "[CH486] Test no data",
			getParams: func(t *testing.T) params {
				return params{
					url:            "https://localhost/ftpusers?page=61&page_size=34",
					userCount:      0,
					expStatus:      http.StatusOK,
					expResultCount: 0,
					expTotalItems:  0,
					expTotalPages:  0,
				}
			},
		},
		{
			name: "[CH487] Test search parameter 1 provided",
			getParams: func(t *testing.T) params {
				return params{
					url:            "https://localhost/ftpusers?q=1",
					userCount:      300,
					expStatus:      http.StatusOK,
					expResultCount: 30,
					expTotalItems:  138,
					expTotalPages:  5,
				}
			},
		},
		{
			name: "[CH487] Test search parameter 1 provided page specified",
			getParams: func(t *testing.T) params {
				return params{
					url:            "https://localhost/ftpusers?page=6&q=1",
					userCount:      300,
					expStatus:      http.StatusOK,
					expResultCount: 0,
					expTotalItems:  138,
					expTotalPages:  5,
				}
			},
		},
		{
			name: "[CH487] Test search parameter 1 provided page and page_size specified",
			getParams: func(t *testing.T) params {
				return params{
					url:            "https://localhost/ftpusers?page=16&page_size=9&q=1",
					userCount:      300,
					expStatus:      http.StatusOK,
					expResultCount: 3,
					expTotalItems:  138,
					expTotalPages:  16,
				}
			},
		},
		{
			name: "[CH487] Test search parameter ZZ provided",
			getParams: func(t *testing.T) params {
				return params{
					url:            "https://localhost/ftpusers?q=zz",
					userCount:      300,
					expStatus:      http.StatusOK,
					expResultCount: 0,
					expTotalItems:  0,
					expTotalPages:  0,
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tParams := test.getParams(t)

			userCount := tParams.userCount
			path := tParams.url
			expResultCount := tParams.expResultCount
			expStatus := tParams.expStatus
			expTotalItems := tParams.expTotalItems
			expTotalPages := tParams.expTotalPages

			u, err := url.Parse(path)
			if err != nil {
				t.Errorf("unexpected error \"%s\" while parsing url", err.Error())
			}

			// get any query string parameters
			qParams := u.Query()

			// get the page parameter if present
			page := 0
			if value, ok := qParams["page"]; ok == true {
				i, err := strconv.ParseInt(value[0], 10, 32)
				if err != nil {
					t.Errorf("unexpected error \"%s\" while parsing value for 'page' parameter", err.Error())
				}
				page = int(i)
			} else {
				page = 1
			}

			// get the page_size parameter if present
			pageSize := 0
			if value, ok := qParams["page_size"]; ok == true {
				i, err := strconv.ParseInt(value[0], 10, 32)
				if err != nil {
					t.Errorf("unexpected error \"%s\" while parsing value for 'page_size' parameter", err.Error())
				}
				pageSize = int(i)
			} else {
				pageSize = 30
			}

			// get the q parameter if present
			search := ""
			if value, ok := qParams["q"]; ok == true {
				search = value[0]
			}

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf(errDBConnectionError, err)
			}
			defer db.Close()

			columns := []string{"id", "username", "description"}

			cntQuery := "select count\\([`\"]id[`\"]\\) from [`\"]ftp_account[`\"]"
			selQuery := "select [`\"]id[`\"], [`\"]username[`\"], [`\"]description[`\"] from [`\"]ftp_account[`\"]"
			searchClause := " where [`\"]username[`\"] like (\\?|\\$1) or [`\"]description[`\"] like (\\?|\\$2)"
			orderClause := " order by [`\"]id[`\"]"

			expPageRows := mock.NewRows(columns)
			var pageData data.FtpUsers

			pageIndex := 0
			pageRow := 0
			searchExists := false
			totalItems := 0
			totalPages := 0
			resultCount := 0
			for r := 1; r <= userCount; r++ {
				username := fmt.Sprintf("myusername%d", r)
				description := fmt.Sprintf("mydescription%d", r)

				if search != "" {
					searchExists = false
					if strings.Contains(username, search) || strings.Contains(description, search) {
						searchExists = true
					}

					if searchExists {
						if pageRow%pageSize == 0 {
							pageIndex++
							pageRow = 0
						}
						pageRow++
						totalItems++
					}
				} else {
					if pageRow%pageSize == 0 {
						pageIndex++
						pageRow = 0
					}
					pageRow++
					totalItems++
				}

				// add the data if it's in the desired page
				if pageIndex == page {
					if search != "" {
						if searchExists {
							expPageRows = expPageRows.AddRow(r, username, description)
							pageData.Ftpusers = append(pageData.Ftpusers, data.FtpUser{ID: uint32(r), Username: username, Description: description, Password: ""})
							resultCount++
						}
					} else {
						expPageRows = expPageRows.AddRow(r, username, description)
						pageData.Ftpusers = append(pageData.Ftpusers, data.FtpUser{ID: uint32(r), Username: username, Description: description, Password: ""})
						resultCount++
					}
				}
			}
			totalPages = totalItems / pageSize
			if totalItems%pageSize > 0 {
				totalPages++
			}

			expCountRows := mock.NewRows([]string{"count(`id`)"})
			expCountRows = expCountRows.AddRow(totalItems)

			pageData.TotalItems = uint32(totalItems)
			pageData.TotalPages = uint32(totalItems / pageSize)
			if totalItems%pageSize > 0 {
				pageData.TotalPages++
			}

			// set the expected queries and their results
			if search != "" {
				mock.ExpectQuery(cntQuery + searchClause).WillReturnRows(expCountRows)
				mock.ExpectQuery(selQuery + searchClause).WillReturnRows(expPageRows)
			} else {
				mock.ExpectQuery(cntQuery).WillReturnRows(expCountRows)
				mock.ExpectQuery(selQuery + orderClause).WillReturnRows(expPageRows)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", path, nil)

			env := Env{Data: &data.Database{DB: db}}

			env.Get(w, r)
			resp := w.Result()

			// check the responses status code
			if resp.StatusCode != expStatus {
				t.Errorf("Expected status %d but received %d", expStatus, resp.StatusCode)
			}

			//get the response returned data
			actual, err := io.ReadAll(resp.Body)
			defer resp.Body.Close()
			if err != nil {
				t.Errorf("unexpected error \"%s\" while reading response body", err.Error())
			}

			var results data.FtpUsers
			err = json.Unmarshal(actual, &results)
			if err != nil {
				t.Errorf("unexpected error \"%s\" while unmarshaling resultant data", err.Error())
			}

			// check the size of the response data
			if len(results.Ftpusers) != expResultCount {
				t.Errorf("the expected %d results returned was not met with %d results", expResultCount, len(results.Ftpusers))
			}

			// check the total items returned
			if results.TotalItems != expTotalItems {
				t.Errorf("the expected %d TotalItems returned was not met with %d TotalItems", expTotalItems, results.TotalItems)
			}

			// check the total pages returned
			if results.TotalPages != expTotalPages {
				t.Errorf("the expected %d TotalPages returned was not met with %d TotalPages", expTotalPages, results.TotalPages)
			}

			// compare each field in the response data to the expected data
			for r := 0; r < resultCount; r++ {
				if pageData.Ftpusers[r].ID != results.Ftpusers[r].ID {
					t.Errorf("expected FtpUser[%d].ID of %d but received %d", r, pageData.Ftpusers[r].ID, results.Ftpusers[r].ID)
				}
				if pageData.Ftpusers[r].Username != results.Ftpusers[r].Username {
					t.Errorf("expected FtpUser[%d].Username of \"%s\" but received \"%s\"", r, pageData.Ftpusers[r].Username, results.Ftpusers[r].Username)
				}
				if pageData.Ftpusers[r].Description != results.Ftpusers[r].Description {
					t.Errorf("expected FtpUser[%d].Description of \"%s\" but received \"%s\"", r, pageData.Ftpusers[r].Description, results.Ftpusers[r].Description)
				}
				if results.Ftpusers[r].Password != "" {
					t.Errorf("expected FtpUser[%d].Password to be empty but received \"%s\"", r, results.Ftpusers[r].Password)
				}
			}

			// serialize the expected data
			expected, err := json.Marshal(pageData)
			if err != nil {
				t.Errorf("unexpected error \"%s\" while marshaling expected data", err.Error())
			}

			// compare the response data to the expected data
			if !bytes.Equal(expected, actual) {
				t.Errorf("the expected json: %s is different from actual %s", expected, actual)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
