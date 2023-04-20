# ftp-user-svc
Microservice to Manage and Validate FTP User Accounts

## Environment Variables
[Configuration](config.md)

## Database Schema
[MySQL DB Schema](schema_mysql.ddl)
[PostgreSQL DB Schema](schema_postgres.ddl)

## Error Response Body
```json
{
      "status": 500,
      "location": "github.com/halt-joe/ftp-user-svc/data/data.FtpUserLookup",
      "message": "An unknown error has occurred",
      "error": "internal server error"
}
```

## Routes
[Routes](routes.md)