package data

import (
	"database/sql"
	"errors"
	"fmt"

	"strings"
	"time"

	sftpgo "github.com/drakkan/sftpgo/v2/dataprovider"
	"github.com/drakkan/sftpgo/v2/kms"
	"github.com/drakkan/sftpgo/v2/vfs"
	"github.com/sftpgo/sdk"
	sdkkms "github.com/sftpgo/sdk/kms"

	// Required by database/sql
	_ "github.com/go-sql-driver/mysql"
	log "github.com/inconshreveable/log15"

	// Required by database/sql
	_ "github.com/lib/pq"
)

// Database - type that will implement Datastore interface
type Database struct {
	*sql.DB
}

// Datastore - interface to the data from the handler environment
type Datastore interface {
	FtpUserLookup(username string) (sftpgo.User, error)
	MappingDelete(system string, id string) (int64, error)
	MappingRetrieve(system string, id string) (Mapping, error)
	MappingCreate(mapping NewMapping) (int, error)
	FtpUserGetSelection(page uint32, pageSize uint32, search string) (FtpUsers, error)
	FtpUserGet(id uint32) (FtpUser, error)
	FtpUserCreate(user FtpUser) (uint32, error)
	FtpUserUpdate(user FtpUser) error
	FtpUserDelete(id uint32) error
	FtpUserUpdatePassword(user FtpUser) error
	SystemIDUserRetrieve(system string) (map[string]string, error)
}

// Custom Errors
const (
	ErrUserNotFound       = "No matching user found"
	ErrMappingNotFound    = "No matching mapping found"
	ErrFTPAccountNotFound = "No matching FTP Account found"
	ErrUnexpectedResult   = "An unexpected result [%d] was returned from a data operation"
	ErrFTPAccountExists   = "An FTP Account for the specified username already exists"
)

// Mapping Create Statuses
const (
	MappingError              = iota
	MappingFTPAccountNotFound = iota
	MappingInserted           = iota
	MappingUpdated            = iota
)

// Database Driver Names
const (
	MySQLDriverName      = "mysql"
	PostgreSQLDriverName = "postgres"
)

// Azure Parameters
var (
	AZKey, AZAccount, AZContainer string
)

// db open connection parameters
var (
	dbDriverName = ""
	connStr      = ""
)

// connection retry constants
const (
	connectionRetryAttempts = 10
	retrySleepSeconds       = 5
)

// FtpUser - type used to contain an FTP User entry
type FtpUser struct {
	ID          uint32 `json:"id,omitempty"`
	Username    string `json:"username,omitempty"`
	Description string `json:"description,omitempty"`
	Password    string `json:"password,omitempty"`
}

// Credentials - type used for checking for the existence of a login
type Credentials struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// Mapping - type used to represent a system, system_id and ftpuser mapping
type Mapping struct {
	System     string  `json:"system,omitempty"`
	ID         string  `json:"id,omitempty"`
	FTPAccount FtpUser `json:"ftp_account,omitempty"`
}

// NewMapping - type used to create a new mapping
type NewMapping struct {
	System       string `json:"system,omitempty"`
	SystemID     string `json:"id,omitempty"`
	FTPAccountID uint32 `json:"ftp_id,omitempty"`
}

// FtpUsers - type used to return a collection of FtpUser structs
type FtpUsers struct {
	Ftpusers   []FtpUser `json:"ftpusers,omitempty"`
	TotalItems uint32    `json:"total_items,omitempty"`
	TotalPages uint32    `json:"total_pages,omitempty"`
}

func fmtQueryForDriver(query string) string {
	result := query

	if dbDriverName == PostgreSQLDriverName {
		result = strings.ReplaceAll(result, "`", "\"")
		for p := 1; p < 11; p++ {
			param := fmt.Sprintf("$%d", p)
			result = strings.Replace(result, "?", param, 1)
		}
	}

	return result
}
func fmtStringParameter(param string) string {
	return strings.ReplaceAll(param, "'", "''")
}
func getPrimaryKeyErr() string {
	result := ""

	if dbDriverName == MySQLDriverName {
		result = "Error Code: 1062"
	}

	if dbDriverName == PostgreSQLDriverName {
		result = "duplicate key value violates unique constraint"
	}

	return result
}
func checkPrimaryKeyErr(err error) bool {
	pkErr := getPrimaryKeyErr()
	return strings.Contains(err.Error(), pkErr)
}
func getForeignKeyErr() string {
	result := ""

	if dbDriverName == MySQLDriverName {
		result = "Error Code: 1452"
	}

	if dbDriverName == PostgreSQLDriverName {
		result = "violates foreign key constraint"
	}

	return result
}
func checkForeignKeyErr(err error) bool {
	fkErr := getForeignKeyErr()

	return strings.Contains(err.Error(), fkErr)
}
func getLimitClauseForDriver(pageSize, offset uint32) string {
	result := ""

	if dbDriverName == MySQLDriverName {
		result = fmt.Sprintf(" limit %d, %d", offset, pageSize)
	}

	if dbDriverName == PostgreSQLDriverName {
		result = fmt.Sprintf(" limit %d offset %d", pageSize, offset)
	}

	return result
}

// attempt to connect to the database with retries <= connectionRetryAttempts
func (db *Database) attemptConnection() error {
	var err error

	for attempt := 0; attempt < connectionRetryAttempts; attempt++ {
		db.DB, err = sql.Open(dbDriverName, connStr)
		if err != nil {
			time.Sleep(retrySleepSeconds * time.Second)
			err = nil
			continue
		}
		break
	}

	if err != nil {
		return err
	}

	// setup connection pooling for PostgreSQL
	if dbDriverName == PostgreSQLDriverName {
		db.SetMaxIdleConns(30)
		db.SetMaxOpenConns(100)
		db.SetConnMaxLifetime(time.Hour)
	}

	return nil
}

// check for a valid db connection
func (db *Database) checkDBConnection() error {
	err := db.Ping()
	if err != nil {
		return db.attemptConnection()
	}
	return nil
}

// NewDB - attempt to connect and return the database
func NewDB(dataSourceName string) (*Database, error) {
	log.Info("Connecting to datasource", "database", dataSourceName)

	segs := strings.Split(dataSourceName, "://")
	if len(segs) < 2 {
		return nil, fmt.Errorf("protocol not specified in %s", dataSourceName)
	}

	if segs[0] == MySQLDriverName {
		dbDriverName = MySQLDriverName
		connStr = segs[1]
	}

	if segs[0] == PostgreSQLDriverName {
		dbDriverName = PostgreSQLDriverName
		connStr = dataSourceName
	}

	if dbDriverName == "" {
		return nil, fmt.Errorf("protocol %s not supported in %s", segs[0], dataSourceName)
	}

	db := Database{}
	err := db.attemptConnection()

	if err != nil {
		return nil, err
	}

	return &db, nil
}

// QueryForDriver - perform the normal Query method after formatting the query based on the driver
func (db *Database) QueryForDriver(query string, args ...interface{}) (*sql.Rows, error) {
	qry := fmtQueryForDriver(query)
	return db.Query(qry, args...)
}

// ExecForDriver - perform the normal Exec method after formatting the query based on the driver
func (db *Database) ExecForDriver(query string, args ...interface{}) (sql.Result, error) {
	qry := fmtQueryForDriver(query)
	return db.Exec(qry, args...)
}

// QueryRowForDriver - perform the normal QueryRow method after formatting the query based on the driver
func (db *Database) QueryRowForDriver(query string, args ...interface{}) *sql.Row {
	qry := fmtQueryForDriver(query)
	return db.QueryRow(qry, args...)
}

// FtpUserLookup - retrieve the FtpUser for the ftp_account entry that corresponds to the supplied username
func (db *Database) FtpUserLookup(username string) (sftpgo.User, error) {
	var user sftpgo.User

	if dbErr := db.checkDBConnection(); dbErr != nil {
		return user, dbErr
	}

	qry := "select a.`id`, a.`username`, a.`description`, a.`password`, m.`id` `folder` "
	qry += "from `ftp_account` a "
	qry += "inner join `ftp_mapping` m "
	qry += "on a.`id` = m.`ftp_id` "
	qry += "where a.`username` = ? "
	qry += "and m.`system` = 'BillSys1'"

	results, err := db.QueryForDriver(qry, username)
	if err != nil {
		return user, err
	}
	defer results.Close()

	userFound := false
	for results.Next() {
		userFound = true
		vf := vfs.VirtualFolder{}

		err = results.Scan(&user.ID, &user.Username, &user.Description, &user.Password, &vf.Name)
		if err != nil {
			return user, err
		}

		vf.VirtualPath = "/" + vf.Name

		vf.FsConfig.Provider = sdk.AzureBlobFilesystemProvider
		vf.FsConfig.AzBlobConfig.AccountName = AZAccount
		vf.FsConfig.AzBlobConfig.Container = AZContainer

		vf.FsConfig.AzBlobConfig.KeyPrefix = vf.Name + "/"
		vf.FsConfig.AzBlobConfig.AccountKey = kms.NewSecret(sdkkms.SecretStatusPlain, AZKey, "", "folder_"+vf.Name)

		user.VirtualFolders = append(user.VirtualFolders, vf)
	}

	// if user has only one virtual folder map it to root
	if len(user.VirtualFolders) == 1 {
		user.FsConfig.Provider = sdk.AzureBlobFilesystemProvider
		user.FsConfig.AzBlobConfig.AccountName = AZAccount
		user.FsConfig.AzBlobConfig.Container = AZContainer

		user.FsConfig.AzBlobConfig.KeyPrefix = user.VirtualFolders[0].Name + "/"
		user.FsConfig.AzBlobConfig.AccountKey = kms.NewSecret(sdkkms.SecretStatusPlain, AZKey, "", "folder_"+user.VirtualFolders[0].Name)
		user.VirtualFolders = nil
	}

	err = results.Err()
	if err != nil {
		log.Error(err.Error())
		return user, err
	}

	if !userFound {
		err = errors.New(ErrUserNotFound)
		return user, err
	}

	return user, nil
}

// MappingDelete - delete the mapping associated with the provided system and systemid
func (db *Database) MappingDelete(system string, id string) (int64, error) {
	if dbErr := db.checkDBConnection(); dbErr != nil {
		return 0, dbErr
	}

	qry := "delete from `ftp_mapping` where `system` = ? and `id` = ?"

	result, err := db.ExecForDriver(qry, system, id)
	if err != nil {
		log.Error(err.Error())
		return 0, err
	}

	rows, err := result.RowsAffected()
	return rows, err
}

// MappingRetrieve - retrieve the mapping associated with the provided system and systemid
func (db *Database) MappingRetrieve(system string, id string) (Mapping, error) {
	var mapping Mapping

	if dbErr := db.checkDBConnection(); dbErr != nil {
		return mapping, dbErr
	}

	mapping.ID = id
	mapping.System = system

	qry := "select a.`id`, a.`username`, a.`description` "
	qry += "from `ftp_mapping` m "
	qry += "inner join `ftp_account` a on m.`ftp_id` = a.`id` "
	qry += "where m.`system` = ? and m.`id` = ?"

	results, err := db.QueryForDriver(qry, system, id)
	if err != nil {
		return mapping, err
	}
	defer results.Close()

	if results.Next() {
		err = results.Scan(&mapping.FTPAccount.ID, &mapping.FTPAccount.Username, &mapping.FTPAccount.Description)
		if err != nil {
			return mapping, err
		}
	} else {
		err = results.Err()
		if err != nil {
			log.Error(err.Error())
			return mapping, err
		}

		err = errors.New(ErrMappingNotFound)
		return mapping, err
	}
	return mapping, nil
}

// MappingCreate - insert a new mapping for the given system, system_id and ftp_id
func (db *Database) MappingCreate(mapping NewMapping) (int, error) {
	if dbErr := db.checkDBConnection(); dbErr != nil {
		return 0, dbErr
	}

	// attempt insert first
	qry := "insert into `ftp_mapping` (`system`, `id`, `ftp_id`) values (?, ?, ?)"

	_, err := db.ExecForDriver(qry, mapping.System, mapping.SystemID, mapping.FTPAccountID)
	if err != nil {
		// if key exists try update
		if checkPrimaryKeyErr(err) {
			qry = "update `ftp_mapping` set `ftp_id` = ? where `system` = ? and `id` = ?"

			_, err = db.ExecForDriver(qry, mapping.FTPAccountID, mapping.System, mapping.SystemID)
			if err != nil {
				if checkForeignKeyErr(err) {
					return MappingFTPAccountNotFound, nil
				}

				return MappingError, err
			}

			return MappingUpdated, nil
		}

		return MappingError, err
	}

	return MappingInserted, nil
}

// FtpUserGetSelection - retrieve all ftp_account entries
// - for the specified page and result set size (default page=1 and page_size=30)
// - a 1 based page index is used
func (db *Database) FtpUserGetSelection(page uint32, pageSize uint32, search string) (users FtpUsers, err error) {
	if err = db.checkDBConnection(); err != nil {
		return
	}

	// set the filter clause if present
	filter := ""
	filterClause := ""
	if search != "" {
		filter = "%" + search + "%"
		filterClause = " where `username` like ? or `description` like ?"
	}

	// get total number of user accounts
	qry := "select count(`id`) from `ftp_account`" + filterClause

	var result *sql.Row
	if search != "" {
		result = db.QueryRowForDriver(qry, filter, filter)
	} else {
		result = db.QueryRowForDriver(qry)
	}

	err = result.Scan(&users.TotalItems)
	if err != nil {
		log.Error(err.Error())
		return users, err
	}

	qry = "select `id`, `username`, `description` from `ftp_account`" + filterClause + " order by `id`"

	// set default page and page_size if not provided
	if pageSize == 0 {
		pageSize = 30
	}
	if page == 0 {
		page = 1
	}

	// set the offset into the ftp_users data set
	offset := (page - 1) * pageSize

	users.TotalPages = users.TotalItems / pageSize
	if users.TotalItems%pageSize > 0 {
		users.TotalPages++
	}

	// add the limit clause to the query and get proper argument order
	qry += getLimitClauseForDriver(pageSize, offset)

	var results *sql.Rows
	if search != "" {
		results, err = db.QueryForDriver(qry, filter, filter)
	} else {
		results, err = db.QueryForDriver(qry)
	}

	if err != nil {
		log.Error(err.Error())
		return users, err
	}
	defer results.Close()

	for results.Next() {
		var user FtpUser
		err = results.Scan(&user.ID, &user.Username, &user.Description)
		if err != nil {
			break
		}
		user.Password = ""
		users.Ftpusers = append(users.Ftpusers, user)
	}

	// check for error from break
	if err != nil {
		log.Error(err.Error())
		return users, err
	}

	err = results.Err()
	if err != nil {
		log.Error(err.Error())
		return users, err
	}

	return users, nil
}

// FtpUserGet - retrieve the ftp_account entry associated with id
func (db *Database) FtpUserGet(id uint32) (FtpUser, error) {
	var user FtpUser

	if dbErr := db.checkDBConnection(); dbErr != nil {
		return user, dbErr
	}

	qry := "select `id`, `username`, `description` from `ftp_account` where `id` = ?"

	results, err := db.QueryForDriver(qry, id)
	if err != nil {
		log.Error(err.Error())
		return user, err
	}
	defer results.Close()

	if results.Next() {
		err = results.Scan(&user.ID, &user.Username, &user.Description)
		if err != nil {
			return user, err
		}
	} else {
		err = results.Err()
		if err != nil {
			log.Error(err.Error())
			return user, err
		}

		err = errors.New(ErrUserNotFound)
		return user, err
	}
	return user, nil
}

// FtpUserCreate - create a ftp_account with the provided parameters
func (db *Database) FtpUserCreate(user FtpUser) (uint32, error) {
	if dbErr := db.checkDBConnection(); dbErr != nil {
		return 0, dbErr
	}

	qry := "insert into `ftp_account` (`username`, `description`, `password`) values (?, ?, ?)"

	_, err := db.ExecForDriver(qry, user.Username, user.Description, user.Password)
	if err != nil {
		if checkPrimaryKeyErr(err) {
			e := errors.New(ErrFTPAccountExists)
			return 0, e
		}
		log.Error(err.Error())
		return 0, err
	}

	var id int
	qry = "select min(`id`) from `ftp_account` where `username` = ?"

	row := db.QueryRowForDriver(qry, user.Username)

	err = row.Scan(&id)
	if err != nil {
		log.Error(err.Error())
		return 0, err
	}

	return uint32(id), nil
}

// FtpUserUpdate - update an ftp_account specified by the ftp user provided
func (db *Database) FtpUserUpdate(user FtpUser) error {
	if dbErr := db.checkDBConnection(); dbErr != nil {
		return dbErr
	}

	qry := "update `ftp_account` set `username` = ?, `description` = ?, `updated_on` = current_timestamp where `id` = ?"

	result, err := db.ExecForDriver(qry, user.Username, user.Description, user.ID)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if rows == 0 {
		e := errors.New(ErrFTPAccountNotFound)
		return e
	}

	return nil
}

// FtpUserDelete - delete the ftp_account specified by the id provided
func (db *Database) FtpUserDelete(id uint32) error {
	if dbErr := db.checkDBConnection(); dbErr != nil {
		return dbErr
	}

	qry := "delete from `ftp_account` where `id` = ?"

	result, err := db.ExecForDriver(qry, id)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if rows == 0 {
		e := errors.New(ErrFTPAccountNotFound)
		return e
	}

	return nil
}

// FtpUserUpdatePassword - update the password on an ftp_account specified by the ftp user provided
func (db *Database) FtpUserUpdatePassword(user FtpUser) error {
	if dbErr := db.checkDBConnection(); dbErr != nil {
		return dbErr
	}

	qry := "update `ftp_account` set `password` = ?, `updated_on` = current_timestamp where `id` = ?"

	result, err := db.ExecForDriver(qry, user.Password, user.ID)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if rows == 0 {
		e := errors.New(ErrFTPAccountNotFound)
		return e
	}

	return nil
}

// SystemIDUserRetrieve - retrieve all of the SystemID and Username
// pairs associated with the provided system
func (db *Database) SystemIDUserRetrieve(system string) (map[string]string, error) {
	result := make(map[string]string)

	if dbErr := db.checkDBConnection(); dbErr != nil {
		return result, dbErr
	}

	qry := "select distinct m.`id`, a.`username` "
	qry += "from `ftp_mapping` m "
	qry += "inner join `ftp_account` a on m.`ftp_id` = a.`id` "
	qry += "where m.`system` = ?"

	results, err := db.QueryForDriver(qry, system)
	if err != nil {
		log.Error(err.Error())
		return result, err
	}
	defer results.Close()

	var id, username string
	for results.Next() {
		err = results.Scan(&id, &username)
		if err != nil {
			return result, err
		}
		result[id] = username
	}

	return result, nil
}
