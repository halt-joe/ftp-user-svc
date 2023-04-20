# ftp-user-svc
Microservice to Manage and Validate FTP User Accounts

## Environment variables
- HTTPPORT The port that the service should listen on -- default: 8080
- DBCON The connection string for the database the service uses
- APIKEY The key used for authenticating clients
- SENTRY_DSN The key and URL for connecting to sentry.  No default, which disables sentry
- SENTRY_ENVIRONMENT The environment the deployment is running in -- no default
- SENTRY_RELEASE The release version -- no default

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
`POST /login`

### Request Body:
```json
{
      "username": "testuser",
      "password": "testpassword"
}
```

### Responses:
- 200 Success
- 401 Unauthorized (Failed Authentication)
- 500 Error

### Response Body:
- 200 Success
```json
{
    "id": 13,
    "status": 1,
    "username": "test-user",
    "password": "test-password",
    "description": "test-description"
}
```

`DELETE /mappings/{system}/{id}`

### Parameters
- system
    the system that the mapping is associated with e.g. "BillSys1
- id
    the id in the {system} mapped to the ftp user

### Responses:
- 204 Successfully Deleted
- 401 Unauthorized (Failed Authentication)
- 404 System ID Not Found
- 500 Error

`GET /mappings/{system}/{id}`

### Parameters
- system
    the system that the mapping is associated with e.g. "BillSys1
- id
    the id in the {system} mapped to the ftp user

### Responses:
- 200 Success
- 401 Unauthorized (Failed Authentication)
- 404 System ID Not Found
- 500 Error

### Response Body:
```json
{
      "system": "BillSys1,
      "id": "999",
      "ftp_account": {
            "id": 11,
            "username":"testuser",
            "description": "test description"
      }
}
```

`POST /mappings/{system}`

### Parameters
- system
    the system that the mapping is associated with e.g. "BillSys1

### Request Body:
```json
{
      "id": "999",
      "ftp_id": 13
}
```

### Responses:
- 200 Updated
- 201 Created
- 400 Bad Request
- 401 Unauthorized (Failed Authentication)
- 404 The requested ftp_id does not exist
- 500 Error

### Response Body:
- 201 Created
```json
{
      "system": "BillSys1,
      "id": "999",
      "ftp_account": {
            "id": 13,
            "username":"testuser",
            "description": "test description"
      }
}
```

`GET /ftpusers`

### Query Parameters
- page (optional)
    - the page index within the ftpusers collection
    - 1 based index
    - default page = 1 if not specified
- page_size (optional)
    - the number of ftpusers to return in the result set
    - default page_size = 30 if not specified
- q (optional)
    - a full text query of the username and description fields

### Responses:
- 200 Success
- 401 Unauthorized (Failed Authentication)
- 500 Error

### Response Body:
```json
{
    "ftpusers": [
      {"id": 11, "username":"testuser", "description": "test description"},
      {"id": 12, "username":"testuser2", "description": "test description 2"},
      ...
      ],
    "total_items": 245,
    "total_pages": 13
}
```

`GET /ftpusers/{id}`

### Parameters:
- id
    the id of the ftp user entry

### Responses:
- 200 Success
- 400 Bad Request
- 401 Unauthorized (Failed Authentication)
- 404 User Not Found
- 500 Error

### Response Body:
```json
      {"id": 11, "username":"testuser", "description": "test description"}
```

`POST /ftpusers`

### Responses:
- 201 Created
- 400 Bad Request
- 401 Unauthorized (Failed Authentication)
- 409 Conflict
- 500 Error

### Request Body
```json
{"username":"myusername", "description":"mydescription", "password":"mypassword"}
```

### Response Body:
```json
{"id": 13, "username":"testuser", "description": "test description"}
```

`PUT /ftpusers/{id}`

### Parameters:
- id
    the id of the ftp user entry

### Request Body
```json
{"username":"myusername", "description":"mydescription"}
```

### Responses:
- 200 Success
- 400 Bad Request
- 401 Unauthorized (Failed Authentication)
- 404 Not Found
- 500 Error

### Response Body:
```json
{"id": 13, "username":"testuser", "description": "test description"}
```

`DELETE /ftpusers/{id}`

### Parameters:
- id
   the id of the ftp user entry

### Responses:
- 204 No Content (Successful Delete)
- 400 Bad Request
- 401 Unauthorized (Failed Authentication)
- 404 User Not Found
- 500 Error

`PATCH /ftpusers/{id}`

### Parameters:
- id
   the id of the ftp user entry

### Responses:
- 200 Success
- 400 Bad Request
- 401 Unauthorized (Failed Authentication)
- 404 Not Found
- 500 Error

`GET /mappings/{system}`

### Parameters
- system
    the system that the system_id and username pairs are associated with e.g. "BillSys1

### Responses:
- 200 Success
- 400 Bad Request
- 401 Unauthorized (Failed Authentication)
- 404 Not Found (System not found)
- 500 Error

### Response Body:
```json
{
      "system_id1": "username1",
      "system_id2": "username2"
}
```