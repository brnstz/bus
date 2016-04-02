// Package conf defines
package conf

// DB is our database config used by both busapi and busloader
type DB struct {

	// DBAddr is the "host:port" we use for connecting to postgres.
	// Default: "localhost:5432"
	// Environment variable: $BUS_DB_ADDR
	DBAddr string `envconfig:"db_addr" default:"localhost:5432"`

	// DBUser is the username we use for connecting to postgres.
	// Default: postgres
	// Environment variable: $BUS_DB_USER
	DBUser string `envconfig:"db_user" default:"postgres"`

	// DBName is the database name we use in postgres.
	// Default: postgres
	// Environment variable: $BUS_DB_NAME
	DBName string `envconfig:"db_name" default:"postgres"`
}

// API is our config spec used by busapi
type API struct {

	// APIAddr is the "host:port" we listen to for incoming HTTP connections.
	// The host can be blank.
	// Default: "0.0.0.0:8000"
	// Environment variable: $BUS_API_ADDR
	APIAddr string `envconfig:"api_addr" default:"0.0.0.0:8000"`

	// RedisAddr is the "host:port" we use for connecting to redis.
	// Default: "localhost:6379"
	// Environment variable: $BUS_REDIS_ADDR
	RedisAddr string `envconfig:"api_addr" default:"localhost:6379"`

	// BusAPIKey is the API key for accessing http://bustime.mta.info/
	// Default: None
	// Environment variable: $BUS_MTA_BUSTIME_API_KEY
	BustimeAPIKey string `envconfig:"mta_bustime_api_key" required:"true"`

	// DatamineAPIKey is the API key for accessing http://datamine.mta.info/
	// Default: None
	// Environment variable: $BUS_MTA_DATAMINE_API_KEY
	DatamineAPIKey string `envconfig:"mta_datamine_api_key" required:"true"`
}

// Loader is our config spec used by busloader
type Loader struct {
	// TmpDir is the root directory we use for creating temporary
	// files when loading new data.
	// Default: None (use system default)
	// Environment variable: $BUS_TMP_DIR
	TmpDir string `envconfig:"tmp_dir"`

	// RouteFilter is a list of route_ids we want
	// to specifically extract from the transit files
	// Default: None (no filter)
	// Environment variable: $BUS_ROUTE_FILTER (comma-delimited list)
	RouteFilter []string `envconfig:route_filter"`

	// GTFSURLs is a pipe-delimited list of URLs that have zipped GTFS
	// feeds. https://developers.google.com/transit/gtfs/
	// Default: None
	// Environment variable: $BUS_GTFS_URLS (comma-delimited list)
	GTFSURLs []string `envconfig:"gtfs_urls"`

	// LoadForever is a boolean that determines whether we just load
	// the GTFS URLs once and exit or whether we continually load them (every
	// 24 hours).
	// Default: true
	// Environment variable: $BUS_LOAD_FOREVER
	LoadForever bool `envconfig:"load_forever" default`
}
