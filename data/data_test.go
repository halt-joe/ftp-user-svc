package data

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	sftpgo "github.com/drakkan/sftpgo/v2/dataprovider"
)

const (
	errDBConnectionError = "an error '%s' was not expected when opening a stub database connection"
)

func TestFtpUserLookup(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errDBConnectionError, err)
	}
	defer db.Close()

	dBase := &Database{db}

	// query := "select [`\"]id[`\"], [`\"]username[`\"], [`\"]description[`\"], [`\"]password[`\"] from [`\"]ftp_account[`\"] where [`\"]username[`\"] = (\\?|\\$1)"
	// columns := []string{"id", "username", "description", "password"}
	query := "select a\\.[`\"]id[`\"], a\\.[`\"]username[`\"], a\\.[`\"]description[`\"], a\\.[`\"]password[`\"], m\\.[`\"]id[`\"] [`\"]folder[`\"] "
	query += "from [`\"]ftp_account[`\"] a "
	query += "inner join [`\"]ftp_mapping[`\"] m "
	query += "on a\\.[`\"]id[`\"] = m\\.[`\"]ftp_id[`\"] "
	query += "where a\\.[`\"]username[`\"] = (\\?|\\$1) "
	query += "and m\\.[`\"]system[`\"] = 'BillSys1'"
	columns := []string{"id", "username", "description", "password", "folder"}

	type params struct {
		username string
		expQuery string
		expRows  *sqlmock.Rows
		expUser  sftpgo.User
		expErr   string
	}
	tests := []struct {
		name      string
		getParams func(t *testing.T) params
	}{
		{
			name: "User Not Found",
			getParams: func(t *testing.T) params {
				expRows := mock.NewRows(columns)
				return params{
					username: "Bad User Name",
					expQuery: query,
					expRows:  expRows,
					// expUser:  "",
					expErr: ErrUserNotFound,
				}
			},
		},
		{
			name: "User Found",
			getParams: func(t *testing.T) params {
				user := sftpgo.User{}
				user.ID = 1
				user.Username = "Test User 1"
				user.Description = "Test Description 1"
				user.Password = "Test Password 1"
				expRows := mock.NewRows(columns)
				expRows = expRows.AddRow(user.ID, user.Username, user.Description, user.Password, "12345")
				return params{
					username: "Test User 1",
					expQuery: query,
					expRows:  expRows,
					expUser:  user,
					// expErr: "",
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tParams := test.getParams(t)

			mock.ExpectQuery(tParams.expQuery).WillReturnRows(tParams.expRows)

			user, err := dBase.FtpUserLookup(tParams.username)
			if err != nil && err.Error() != tParams.expErr {
				t.Errorf("unexpected error from FtpUserLookup %s", err)
			}
			if err == nil && tParams.expErr != "" {
				t.Errorf("expected error not returned from FtpUserLookup")
			}
			if user.Username != tParams.expUser.Username {
				t.Errorf("unexpected Username returned %s expected %s", user.Username, tParams.expUser.Username)
			}
			if user.ID != tParams.expUser.ID {
				t.Errorf("unexpected ID returned %d expected %d", user.ID, tParams.expUser.ID)
			}
			if user.Description != tParams.expUser.Description {
				t.Errorf("unexpected Description returned %s expected %s", user.Description, tParams.expUser.Description)
			}
			if user.Password != tParams.expUser.Password {
				t.Errorf("unexpected Password returned %s expected %s", user.Password, tParams.expUser.Password)
			}
			if user.ID != 0 && tParams.username != user.Username {
				t.Errorf("returned username %s does not match passed in username %s", user.Username, tParams.username)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
func TestMappingDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errDBConnectionError, err)
	}
	defer db.Close()

	dBase := &Database{db}

	query := "delete from [`\"]ftp_mapping[`\"] where [`\"]system[`\"] = (\\?|\\$1) and [`\"]id[`\"] = (\\?|$2)"

	type params struct {
		system      string
		id          string
		expQuery    string
		expResult   sql.Result
		expRowCount int64
		expErr      string
	}
	tests := []struct {
		name      string
		getParams func(t *testing.T) params
	}{
		{
			name: "Mapping Not Found",
			getParams: func(t *testing.T) params {
				return params{
					system:      "Bad System",
					id:          "Bad System ID",
					expQuery:    query,
					expResult:   sqlmock.NewResult(0, 0),
					expRowCount: 0,
					// expErr: "",
				}
			},
		},
		{
			name: "Mapping Deleted",
			getParams: func(t *testing.T) params {
				return params{
					system:      "Good System",
					id:          "Good System ID",
					expQuery:    query,
					expResult:   sqlmock.NewResult(0, 1),
					expRowCount: 1,
					// expErr: "",
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tParams := test.getParams(t)

			mock.ExpectExec(tParams.expQuery).WillReturnResult(tParams.expResult)

			rowcount, err := dBase.MappingDelete(tParams.system, tParams.id)
			if err != nil && err.Error() != tParams.expErr {
				t.Errorf("unexpected error from MappingDelete %s", err)
			}
			if err == nil && tParams.expErr != "" {
				t.Errorf("expected error not returned from MappingDelete")
			}
			if rowcount != tParams.expRowCount {
				t.Errorf("unexpected rowcount of %d returned.  expected %d", rowcount, tParams.expRowCount)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestMappingRetrieve(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errDBConnectionError, err)
	}
	defer db.Close()

	dBase := &Database{db}

	columns := []string{"id", "username", "description"}

	query := "select a.[`\"]id[`\"], a.[`\"]username[`\"], a.[`\"]description[`\"] "
	query += "from [`\"]ftp_mapping[`\"] m "
	query += "inner join [`\"]ftp_account[`\"] a on m.[`\"]ftp_id[`\"] = a.[`\"]id[`\"] "
	query += "where m.[`\"]system[`\"] = (\\?|\\$1) and m.[`\"]id[`\"] = (\\?|\\$2)"

	type params struct {
		system     string
		id         string
		expQuery   string
		expRows    *sqlmock.Rows
		expMapping Mapping
		expErr     string
	}
	tests := []struct {
		name      string
		getParams func(t *testing.T) params
	}{
		{
			name: "Mapping Not Found",
			getParams: func(t *testing.T) params {
				mapping := Mapping{"Bad System", "Bad System ID", FtpUser{}}
				expRows := mock.NewRows(columns)
				return params{
					system:     mapping.System,
					id:         mapping.ID,
					expQuery:   query,
					expRows:    expRows,
					expMapping: mapping,
					expErr:     ErrMappingNotFound,
				}
			},
		},
		{
			name: "Mapping Found",
			getParams: func(t *testing.T) params {
				user := FtpUser{1, "Good User 1", "Good Description 1", ""}
				mapping := Mapping{"Good System", "Good System ID", user}
				expRows := mock.NewRows(columns)
				expRows = expRows.AddRow(user.ID, user.Username, user.Description)
				return params{
					system:     mapping.System,
					id:         mapping.ID,
					expQuery:   query,
					expRows:    expRows,
					expMapping: mapping,
					expErr:     "",
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tParams := test.getParams(t)

			mock.ExpectQuery(tParams.expQuery).WillReturnRows(tParams.expRows)

			mapping, err := dBase.MappingRetrieve(tParams.system, tParams.id)
			if err != nil && err.Error() != tParams.expErr {
				t.Errorf("unexpected error from MappingRetrieve %s", err)
			}
			if err == nil && tParams.expErr != "" {
				t.Errorf("expected error not returned from MappingRetrieve")
			}
			if mapping.FTPAccount.ID != tParams.expMapping.FTPAccount.ID {
				t.Errorf("unexpected FTPAccount.ID returned %d expected %d", mapping.FTPAccount.ID, tParams.expMapping.FTPAccount.ID)
			}
			if mapping.FTPAccount.Username != tParams.expMapping.FTPAccount.Username {
				t.Errorf("unexpected FTPAccount.Username returned %s expected %s", mapping.FTPAccount.Username, tParams.expMapping.FTPAccount.Username)
			}
			if mapping.FTPAccount.Description != tParams.expMapping.FTPAccount.Description {
				t.Errorf("unexpected FTPAccount.Description returned %s expected %s", mapping.FTPAccount.Description, tParams.expMapping.FTPAccount.Description)
			}
			if mapping.FTPAccount.Password != "" {
				t.Errorf("unexpected password of %s returned expecting \"\"", mapping.FTPAccount.Password)
			}
			if mapping.System != tParams.expMapping.System {
				t.Errorf("unexpected System returned %s expected %s", mapping.System, tParams.expMapping.System)
			}
			if mapping.ID != tParams.expMapping.ID {
				t.Errorf("unexpected ID returned %s expected %s", mapping.ID, tParams.expMapping.ID)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
func TestMappingCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errDBConnectionError, err)
	}
	defer db.Close()

	dBase := &Database{db}

	insQuery := "insert into [`\"]ftp_mapping[`\"] \\([`\"]system[`\"], [`\"]id[`\"], [`\"]ftp_id[`\"]\\) values \\((\\?|\\$1), (\\?|\\$2), (\\?|\\$3)\\)"
	updQuery := "update [`\"]ftp_mapping[`\"] set [`\"]ftp_id[`\"] = (\\?|\\$1) where [`\"]system[`\"] = (\\?|\\$2) and [`\"]id[`\"] = (\\?|\\$3)"

	type params struct {
		newmapping NewMapping
		expQueries []string
		expArgs    [][]driver.Value
		expResults []sql.Result
		expErrors  []error
		expStatus  int
		expErr     string
	}
	tests := []struct {
		name      string
		getParams func(t *testing.T) params
	}{
		{
			name: "Insert Successful",
			getParams: func(t *testing.T) params {
				mapping := NewMapping{"Good System", "Good System ID", 1}
				return params{
					newmapping: mapping,
					expQueries: []string{insQuery},
					expArgs:    [][]driver.Value{{mapping.System, mapping.SystemID, mapping.FTPAccountID}},
					expResults: []sql.Result{sqlmock.NewResult(0, 1)},
					// expErrors:  []error{},
					expStatus: MappingInserted,
					// expErr:     "",
				}
			},
		},
		{
			name: "Update Successful",
			getParams: func(t *testing.T) params {
				mapping := NewMapping{"Good System", "Good System ID", 1}
				return params{
					newmapping: mapping,
					expQueries: []string{insQuery, updQuery},
					expArgs:    [][]driver.Value{{mapping.System, mapping.SystemID, mapping.FTPAccountID}, {mapping.FTPAccountID, mapping.System, mapping.SystemID}},
					expResults: []sql.Result{sqlmock.NewResult(0, 0), sqlmock.NewResult(0, 1)},
					expErrors:  []error{errors.New(getPrimaryKeyErr())},
					expStatus:  MappingUpdated,
					// expErr:     "",
				}
			},
		},
		{
			name: "FTPAccountID Doesn't Exist",
			getParams: func(t *testing.T) params {
				mapping := NewMapping{"Bad System", "Bad System ID", 1}
				return params{
					newmapping: mapping,
					expQueries: []string{insQuery, updQuery},
					expArgs:    [][]driver.Value{{mapping.System, mapping.SystemID, mapping.FTPAccountID}, {mapping.FTPAccountID, mapping.System, mapping.SystemID}},
					expResults: []sql.Result{sqlmock.NewResult(0, 0), sqlmock.NewResult(0, 0)},
					expErrors:  []error{errors.New(getPrimaryKeyErr()), errors.New(getForeignKeyErr())},
					expStatus:  MappingFTPAccountNotFound,
					// expErr:     "",
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tParams := test.getParams(t)

			for q := 0; q < len(tParams.expQueries); q++ {
				ex := mock.ExpectExec(tParams.expQueries[q])
				ex.WithArgs(tParams.expArgs[q]...)
				ex.WillReturnResult(tParams.expResults[q])
				if q < len(tParams.expErrors) {
					ex.WillReturnError(tParams.expErrors[q])
				}
			}

			status, err := dBase.MappingCreate(tParams.newmapping)

			if err != nil && err.Error() != tParams.expErr {
				t.Errorf("unexpected error from MappingCreate %s", err)
			}
			if err == nil && tParams.expErr != "" {
				t.Errorf("expected error not returned from MappingCreate")
			}
			if status != tParams.expStatus {
				t.Errorf("expected status of %d but received %d", tParams.expStatus, status)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
func TestFtpUserGetSelection(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errDBConnectionError, err)
	}
	defer db.Close()

	dBase := &Database{db}

	cntColumns := []string{"count"}
	selColumns := []string{"id", "username", "description"}
	cntQuery := "select count\\([`\"]id[`\"]\\) from [`\"]ftp_account[`\"]"
	selQuery := "select [`\"]id[`\"], [`\"]username[`\"], [`\"]description[`\"] from [`\"]ftp_account[`\"]"
	searchClause := " where [`\"]username[`\"] like (\\?|\\$1) or [`\"]description[`\"] like (\\?|\\$2)"
	orderClause := " order by [`\"]id[`\"]"

	type params struct {
		page       uint32
		pageSize   uint32
		search     string
		expLimit   string
		expQueries []string
		expRows    []*sqlmock.Rows
		expErrors  []error
		expUsers   FtpUsers
		expErr     string
	}

	tests := []struct {
		name      string
		getParams func(t *testing.T) params
	}{
		{
			name: "Test Page 0 Size 0 Search Test",
			getParams: func(t *testing.T) params {
				userCount := 30
				cntRows := sqlmock.NewRows(cntColumns)
				cntRows = cntRows.AddRow(userCount)
				selRows := sqlmock.NewRows(selColumns)
				var users FtpUsers
				for r := 1; r <= userCount; r++ {
					user := fmt.Sprintf("Test User %d", r)
					desc := fmt.Sprintf("Test Description %d", r)
					selRows = selRows.AddRow(r, user, desc)

					users.Ftpusers = append(users.Ftpusers, FtpUser{uint32(r), user, desc, ""})
				}
				lPageSize := uint32(30)
				lPage := uint32(1)
				lOffset := (lPage - 1) * lPageSize

				users.TotalItems = uint32(len(users.Ftpusers))
				users.TotalPages = users.TotalItems / uint32(lPageSize)
				if users.TotalItems%uint32(lPageSize) > 0 {
					users.TotalPages++
				}

				return params{
					page:       0,
					pageSize:   0,
					search:     "Test",
					expLimit:   getLimitClauseForDriver(lPageSize, lOffset),
					expQueries: []string{cntQuery, selQuery},
					expRows:    []*sqlmock.Rows{cntRows, selRows},
					expErrors:  []error{},
					expUsers:   users,
					expErr:     "",
				}

			},
		},
		{
			name: "Test Page 5 Size 0 Search",
			getParams: func(t *testing.T) params {
				userCount := 5
				cntRows := sqlmock.NewRows(cntColumns)
				cntRows = cntRows.AddRow(userCount)
				selRows := sqlmock.NewRows(selColumns)
				var users FtpUsers
				for r := 1; r <= userCount; r++ {
					user := fmt.Sprintf("Test User %d", r)
					desc := fmt.Sprintf("Test Description %d", r)
					selRows = selRows.AddRow(r, user, desc)

					users.Ftpusers = append(users.Ftpusers, FtpUser{uint32(r), user, desc, ""})
				}
				lPageSize := uint32(30)
				lPage := uint32(5)
				lOffset := (lPage - 1) * lPageSize

				users.TotalItems = uint32(len(users.Ftpusers))
				users.TotalPages = users.TotalItems / uint32(lPageSize)
				if users.TotalItems%uint32(lPageSize) > 0 {
					users.TotalPages++
				}

				return params{
					page:       5,
					pageSize:   0,
					search:     "",
					expLimit:   getLimitClauseForDriver(lPageSize, lOffset),
					expQueries: []string{cntQuery, selQuery},
					expRows:    []*sqlmock.Rows{cntRows, selRows},
					expErrors:  []error{},
					expUsers:   users,
					expErr:     "",
				}

			},
		},
		{
			name: "Test Page 5 Size 3 Search Test",
			getParams: func(t *testing.T) params {
				userCount := 3
				cntRows := sqlmock.NewRows(cntColumns)
				cntRows = cntRows.AddRow(userCount)
				selRows := sqlmock.NewRows(selColumns)
				var users FtpUsers
				for r := 1; r <= userCount; r++ {
					user := fmt.Sprintf("Test User %d", r)
					desc := fmt.Sprintf("Test Description %d", r)
					selRows = selRows.AddRow(r, user, desc)

					users.Ftpusers = append(users.Ftpusers, FtpUser{uint32(r), user, desc, ""})
				}
				lPageSize := uint32(3)
				lPage := uint32(5)
				lOffset := (lPage - 1) * lPageSize

				users.TotalItems = uint32(len(users.Ftpusers))
				users.TotalPages = users.TotalItems / uint32(lPageSize)
				if users.TotalItems%uint32(lPageSize) > 0 {
					users.TotalPages++
				}

				return params{
					page:       5,
					pageSize:   3,
					search:     "Test",
					expLimit:   getLimitClauseForDriver(lPageSize, lOffset),
					expQueries: []string{cntQuery, selQuery},
					expRows:    []*sqlmock.Rows{cntRows, selRows},
					expErrors:  []error{},
					expUsers:   users,
					expErr:     "",
				}

			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tParams := test.getParams(t)

			for q := 0; q < len(tParams.expQueries); q++ {
				srch := ""
				fltr := ""
				if tParams.search != "" {
					fltr = "%" + tParams.search + "%"
					srch = searchClause
				}

				lmt := ""
				ord := ""
				if tParams.expQueries[q] == selQuery {
					lmt = tParams.expLimit
					ord = orderClause
				}
				ex := mock.ExpectQuery(tParams.expQueries[q] + srch + ord + lmt)
				if srch != "" {
					ex.WithArgs(fltr, fltr)
				}
				ex.WillReturnRows(tParams.expRows[q])
				if q < len(tParams.expErrors) {
					ex.WillReturnError(tParams.expErrors[q])
				}
			}
			users, err := dBase.FtpUserGetSelection(tParams.page, tParams.pageSize, tParams.search)
			if err != nil && err.Error() != tParams.expErr {
				t.Errorf("unexpected error from FtpUserGetSelection %s", err)
			}
			if err == nil && tParams.expErr != "" {
				t.Errorf("expected error not returned from FtpUserGetSelection")
			}
			if len(users.Ftpusers) != len(tParams.expUsers.Ftpusers) {
				t.Errorf("%d ftpusers returned.  expected %d", len(users.Ftpusers), len(tParams.expUsers.Ftpusers))
			}
			for u := 0; u < len(users.Ftpusers); u++ {
				r := users.Ftpusers[u]
				e := tParams.expUsers.Ftpusers[u]
				if r.ID != e.ID {
					t.Errorf("Expected ID %d for user %s was not met with %d", e.ID, e.Username, r.ID)
				}
				if r.Username != e.Username {
					t.Errorf("Expected Username %s for user %s was not met with %s", e.Username, e.Username, r.Username)
				}
				if r.Description != e.Description {
					t.Errorf("Expected Description %s for user %s was not met with %s", e.Description, e.Username, r.Description)
				}
				if r.Password != "" {
					t.Errorf("Unexpected password %s returned for user %s", r.Password, e.Username)
				}
			}
			if users.TotalItems != tParams.expUsers.TotalItems {
				t.Errorf("expected TotalItems %d was not met with %d", users.TotalItems, tParams.expUsers.TotalItems)
			}
			if users.TotalPages != tParams.expUsers.TotalPages {
				t.Errorf("expected TotalPages %d was not met with %d", users.TotalPages, tParams.expUsers.TotalPages)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
func TestFtpUserGet(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errDBConnectionError, err)
	}
	defer db.Close()

	dBase := &Database{db}

	selColumns := []string{"id", "username", "description"}
	selQuery := "select [`\"]id[`\"], [`\"]username[`\"], [`\"]description[`\"] from [`\"]ftp_account[`\"] where [`\"]id[`\"] = (\\?|\\$1)"

	type params struct {
		id       uint32
		expQuery string
		expRows  *sqlmock.Rows
		expUser  FtpUser
		expErr   string
	}

	tests := []struct {
		name      string
		getParams func(t *testing.T) params
	}{
		{
			name: "User Not Found",
			getParams: func(t *testing.T) params {
				user := FtpUser{}
				userRows := sqlmock.NewRows(selColumns)
				return params{
					id:       1,
					expQuery: selQuery,
					expRows:  userRows,
					expUser:  user,
					expErr:   ErrUserNotFound,
				}
			},
		},
		{
			name: "User Found",
			getParams: func(t *testing.T) params {
				user := FtpUser{1, "Test User 1", "Test Description 1", ""}
				userRows := sqlmock.NewRows(selColumns)
				userRows = userRows.AddRow(user.ID, user.Username, user.Description)
				return params{
					id:       1,
					expQuery: selQuery,
					expRows:  userRows,
					expUser:  user,
					expErr:   "",
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tParams := test.getParams(t)

			ex := mock.ExpectQuery(tParams.expQuery)
			ex.WithArgs(tParams.id)
			ex.WillReturnRows(tParams.expRows)

			r, err := dBase.FtpUserGet(tParams.id)
			if err != nil && err.Error() != tParams.expErr {
				t.Errorf("unexpected error from FtpUserGet %s", err)
			}
			if err == nil && tParams.expErr != "" {
				t.Errorf("expected error not returned from FtpUserGet")
			}

			e := tParams.expUser
			if r.ID != e.ID {
				t.Errorf("Expected ID %d for user %s was not met with %d", e.ID, e.Username, r.ID)
			}
			if r.Username != e.Username {
				t.Errorf("Expected Username %s for user %s was not met with %s", e.Username, e.Username, r.Username)
			}
			if r.Description != e.Description {
				t.Errorf("Expected Description %s for user %s was not met with %s", e.Description, e.Username, r.Description)
			}
			if r.Password != "" {
				t.Errorf("Unexpected password %s returned for user %s", r.Password, e.Username)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
func TestFtpUserCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errDBConnectionError, err)
	}
	defer db.Close()

	dBase := &Database{db}

	insQuery := "insert into [`\"]ftp_account[`\"] \\([`\"]username[`\"], [`\"]description[`\"], [`\"]password[`\"]\\) values \\((\\?|\\$1), (\\?|\\$2), (\\?|\\$3)\\)"
	selColumns := []string{"min"}
	selQuery := "select min\\([`\"]id[`\"]\\) from [`\"]ftp_account[`\"] where [`\"]username[`\"] = (\\?|\\$1)"

	type params struct {
		user       FtpUser
		expQueries []string
		expResult  sql.Result
		expRows    *sqlmock.Rows
		expError   error
		expID      uint32
		expErr     string
	}

	tests := []struct {
		name      string
		getParams func(t *testing.T) params
	}{
		{
			name: "User Account Exists",
			getParams: func(t *testing.T) params {
				return params{
					user:       FtpUser{},
					expQueries: []string{insQuery},
					expResult:  sqlmock.NewResult(0, 0),
					expRows:    nil,
					expError:   errors.New(ErrFTPAccountExists),
					expID:      0,
					expErr:     ErrFTPAccountExists,
				}
			},
		},
		{
			name: "User Account Created",
			getParams: func(t *testing.T) params {
				user := FtpUser{1, "Test User 1", "Test Description 1", "Test Password 1"}
				minRows := sqlmock.NewRows(selColumns)
				minRows = minRows.AddRow(user.ID)
				return params{
					user:       user,
					expQueries: []string{insQuery, selQuery},
					expResult:  sqlmock.NewResult(0, 1),
					expRows:    minRows,
					expError:   nil,
					expID:      1,
					expErr:     "",
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tParams := test.getParams(t)
			for q := 0; q < len(tParams.expQueries); q++ {
				if tParams.expQueries[q] == insQuery {
					ex := mock.ExpectExec(tParams.expQueries[q])
					ex.WithArgs(tParams.user.Username, tParams.user.Description, tParams.user.Password)
					ex.WillReturnResult(tParams.expResult)
					ex.WillReturnError(tParams.expError)
				}
				if tParams.expQueries[q] == selQuery {
					ex := mock.ExpectQuery(tParams.expQueries[q])
					ex.WithArgs(tParams.user.Username)
					ex.WillReturnRows(tParams.expRows)
				}
			}

			id, err := dBase.FtpUserCreate(tParams.user)
			if err != nil && err.Error() != tParams.expErr {
				t.Errorf("unexpected error from FtpUserCreate %s", err)
			}
			if err == nil && tParams.expErr != "" {
				t.Errorf("expected error not returned from FtpUserCreate")
			}

			if id != tParams.expID {
				t.Errorf("Expected ID %d for user %s was not met with %d", tParams.expID, tParams.user.Username, id)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
func TestFtpUserUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errDBConnectionError, err)
	}
	defer db.Close()

	dBase := &Database{db}

	updQuery := "update [`\"]ftp_account[`\"] set [`\"]username[`\"] = (\\?|\\$1), [`\"]description[`\"] = (\\?|\\$2), [`\"]updated_on[`\"] = current_timestamp where [`\"]id[`\"] = (\\?|\\$3)"

	type params struct {
		user      FtpUser
		expQuery  string
		expResult sql.Result
		expErr    string
	}

	tests := []struct {
		name      string
		getParams func(t *testing.T) params
	}{
		{
			name: "Account Not Found",
			getParams: func(t *testing.T) params {
				return params{
					user:      FtpUser{},
					expQuery:  updQuery,
					expResult: sqlmock.NewResult(0, 0),
					expErr:    ErrFTPAccountNotFound,
				}
			},
		},
		{
			name: "Account Updated",
			getParams: func(t *testing.T) params {
				user := FtpUser{1, "Test User 1", "Test Description 1", ""}
				return params{
					user:      user,
					expQuery:  updQuery,
					expResult: sqlmock.NewResult(0, 1),
					expErr:    "",
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tParams := test.getParams(t)

			ex := mock.ExpectExec(tParams.expQuery)
			ex.WithArgs(tParams.user.Username, tParams.user.Description, tParams.user.ID)
			ex.WillReturnResult(tParams.expResult)

			err := dBase.FtpUserUpdate(tParams.user)
			if err != nil && err.Error() != tParams.expErr {
				t.Errorf("unexpected error from FtpUserUpdate %s", err)
			}
			if err == nil && tParams.expErr != "" {
				t.Errorf("expected error not returned from FtpUserUpdate")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
func TestFtpUserDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errDBConnectionError, err)
	}
	defer db.Close()

	dBase := &Database{db}

	delQuery := "delete from [`\"]ftp_account[`\"] where [`\"]id[`\"] = (\\?|\\$1)"

	type params struct {
		id        uint32
		expQuery  string
		expResult sql.Result
		expErr    string
	}

	tests := []struct {
		name      string
		getParams func(t *testing.T) params
	}{
		{
			name: "Account Not Found",
			getParams: func(t *testing.T) params {
				return params{
					id:        1,
					expQuery:  delQuery,
					expResult: sqlmock.NewResult(0, 0),
					expErr:    ErrFTPAccountNotFound,
				}
			},
		},
		{
			name: "Account Deleted",
			getParams: func(t *testing.T) params {
				return params{
					id:        1,
					expQuery:  delQuery,
					expResult: sqlmock.NewResult(0, 1),
					expErr:    "",
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tParams := test.getParams(t)

			ex := mock.ExpectExec(tParams.expQuery)
			ex.WithArgs(tParams.id)
			ex.WillReturnResult(tParams.expResult)

			err := dBase.FtpUserDelete(tParams.id)
			if err != nil && err.Error() != tParams.expErr {
				t.Errorf("unexpected error from FtpUserDelete %s", err)
			}
			if err == nil && tParams.expErr != "" {
				t.Errorf("expected error not returned from FtpUserDelete")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
func TestFtpUserUpdatePassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errDBConnectionError, err)
	}
	defer db.Close()

	dBase := &Database{db}

	updQuery := "update [`\"]ftp_account[`\"] set [`\"]password[`\"] = (\\?|\\$1), [`\"]updated_on[`\"] = current_timestamp where [`\"]id[`\"] = (\\?|\\$1)"

	type params struct {
		user      FtpUser
		expQuery  string
		expResult sql.Result
		expErr    string
	}

	tests := []struct {
		name      string
		getParams func(t *testing.T) params
	}{
		{
			name: "Account Not Found",
			getParams: func(t *testing.T) params {
				return params{
					user:      FtpUser{1, "", "", "New Password"},
					expQuery:  updQuery,
					expResult: sqlmock.NewResult(0, 0),
					expErr:    ErrFTPAccountNotFound,
				}
			},
		},
		{
			name: "Password Updated",
			getParams: func(t *testing.T) params {
				return params{
					user:      FtpUser{1, "", "", "New Password"},
					expQuery:  updQuery,
					expResult: sqlmock.NewResult(0, 1),
					expErr:    "",
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tParams := test.getParams(t)

			ex := mock.ExpectExec(tParams.expQuery)
			ex.WithArgs(tParams.user.Password, tParams.user.ID)
			ex.WillReturnResult(tParams.expResult)

			err := dBase.FtpUserUpdatePassword(tParams.user)
			if err != nil && err.Error() != tParams.expErr {
				t.Errorf("unexpected error from FtpUserUpdatePassword %s", err)
			}
			if err == nil && tParams.expErr != "" {
				t.Errorf("expected error not returned from FtpUserUpdatePassword")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}

		})
	}
}
func TestSystemIDUserRetrieve(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(errDBConnectionError, err)
	}
	defer db.Close()

	dBase := &Database{db}

	query := "select distinct m\\.[`\"]id[`\"], a\\.[`\"]username[`\"] "
	query += "from [`\"]ftp_mapping[`\"] m "
	query += "inner join [`\"]ftp_account[`\"] a on m\\.[`\"]ftp_id[`\"] = a\\.[`\"]id[`\"] "
	query += "where m\\.[`\"]system[`\"] = (\\?|\\$1)"

	columns := []string{"id", "username"}

	type params struct {
		system   string
		expQuery string
		expRows  *sqlmock.Rows
		expPairs map[string]string
	}
	tests := []struct {
		name      string
		getParams func(t *testing.T) params
	}{
		{
			name: "System Not Found",
			getParams: func(t *testing.T) params {
				expRows := mock.NewRows(columns)
				return params{
					system:   "BillSys1",
					expQuery: query,
					expRows:  expRows,
					expPairs: map[string]string{},
				}
			},
		},
		{
			name: "System Pairs Returned",
			getParams: func(t *testing.T) params {
				expRows := mock.NewRows(columns)
				expRows = expRows.AddRow("system_id1", "username1")
				expRows = expRows.AddRow("system_id2", "username2")
				return params{
					system:   "BillSys1",
					expQuery: query,
					expRows:  expRows,
					expPairs: map[string]string{"system_id1": "username1", "system_id2": "username2"},
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tParams := test.getParams(t)

			ex := mock.ExpectQuery(tParams.expQuery)
			ex.WithArgs(tParams.system)
			ex.WillReturnRows(tParams.expRows)

			results, err := dBase.SystemIDUserRetrieve(tParams.system)
			if err != nil {
				t.Errorf("unexpected error from SystemIDUserRetrieve %s", err)
			}

			if reflect.DeepEqual(results, tParams.expPairs) == false {
				t.Error("the returned results did not match the expected results")
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}

		})
	}
}
