# Configuration

Configuration is done with environment variables.  Not all settings have default values.  If there's no value set the feature my be disabled.  

## Environment Variables

Variable | Default | Description |
-------  | ------- | -----------
HTTPPORT | 8080 | The port that the service should listen on
DBCON |  | The connection string for the database the service uses
APIKEY |  | The key used for authenticating clients
SENTRY_DSN |  | The key and URL for connecting to sentry.  No default, which disables sentry
SENTRY_ENVIRONMENT |  | The environment the deployment is running in
SENTRY_RELEASE |  | The release version
AZACCOUNT | | The azure blob storage account
AZKEY | | The azure blob storage key associated with the account
AZCONTAINER | | The azure blob storage container to be used with the account
