// Package conf defines config structs with struct tags as expected
// by https://github.com/kelseyhightower/envconfig
package conf

var (
	// DB is current database config
	DB DBSpec

	// API is the current API config
	API APISpec

	// Loader is the current loader config
	Loader LoaderSpec
)

// DBSpec is our database config used by both busapi and busloader
type DBSpec struct {

	// DBAddr is the "host:port" we use for connecting to postgres
	// Default: "localhost:5432"
	// Environment variable: $BUS_DB_ADDR
	Addr string `envconfig:"db_addr" default:"localhost:5432"`

	// DBUser is the username we use for connecting to postgres
	// Default: postgres
	// Environment variable: $BUS_DB_USER
	User string `envconfig:"db_user" default:"postgres"`

	// DBName is the database name we use in postgres
	// Default: postgres
	// Environment variable: $BUS_DB_NAME
	Name string `envconfig:"db_name" default:"postgres"`
}

// APISpec is our config spec used by busapi
type APISpec struct {

	// APIAddr is the "host:port" we listen to for incoming HTTP connections
	// Default: "0.0.0.0:8000"
	// Environment variable: $BUS_API_ADDR
	Addr string `envconfig:"api_addr" default:"0.0.0.0:8000"`

	// RedisAddr is the "host:port" we use for connecting to redis
	// Default: "localhost:6379"
	// Environment variable: $BUS_REDIS_ADDR
	RedisAddr string `envconfig:"redis_addr" default:"localhost:6379"`

	// BusAPIKey is the API key for accessing http://bustime.mta.info/
	// Default: None
	// Environment variable: $BUS_MTA_BUSTIME_API_KEY
	BustimeAPIKey string `envconfig:"mta_bustime_api_key" required:"true"`

	// DatamineAPIKey is the API key for accessing http://datamine.mta.info/
	// Default: None
	// Environment variable: $BUS_MTA_DATAMINE_API_KEY
	DatamineAPIKey string `envconfig:"mta_datamine_api_key" required:"true"`

	// WebDir is the location of the static web assets
	// Default: ../../web/dist
	// Environment variable: $BUS_WEB_DIR
	WebDir string `envconfig:"web_dir" default:"../../web/dist"`

	// BuildTimestamp is the UNIX timestamp when static assets were last built.
	// Default: 0 (API will use current timestamp)
	// Environment variable: $BUS_BUILD_TIMESTAMP
	BuildTimestamp int64 `envconfig:"build_timestamp"`
}

// LoaderSpec is our config spec used by busloader
type LoaderSpec struct {
	// TmpDir is the root directory we use for creating temporary
	// files when loading new data
	// Default: None (use system default)
	// Environment variable: $BUS_TMP_DIR
	TmpDir string `envconfig:"tmp_dir"`

	// RouteFilter is a comma-delimited list of route_ids we want to
	// specifically extract from the transit files
	// Default: None (no filter)
	// Environment variable: $BUS_ROUTE_FILTER (comma-delimited list)
	RouteFilter []string `envconfig:"route_filter"`

	// GTFSURLs is a comma-delimited list of URLs that have zipped GTFS
	// feeds, see: https://developers.google.com/transit/gtfs/
	// Default: None
	// Environment variable: $BUS_GTFS_URLS (comma-delimited list)
	GTFSURLs []string `envconfig:"gtfs_urls"`

	// LoadForever is a boolean that determines whether we just load
	// the GTFS URLs once and exit or whether we continually load them (every
	// 24 hours)
	// Default: true
	// Environment variable: $BUS_LOAD_FOREVER
	LoadForever bool `envconfig:"load_forever" default:"false"`
}
