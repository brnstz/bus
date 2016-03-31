package conf

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
)

var (
	// RedisTTL is how many seconds we cache things in redis
	RedisTTL = 30

	// RedisConnectTimeout is how long we wait to connect to redis
	// before giving up
	RedisConnectTimeout = 1 * time.Second

	// DB is our shared connection to postgres
	DB *sqlx.DB

	// Incoming options from environment

	// APIAddr is the "host:port" we listen to for incoming HTTP connections.
	// The host can be blank.
	// Default: ":8000"
	// Environment variable: $BUS_API_ADDR
	APIAddr string

	// DBAddr is the "host:port" we use for connecting to postgres.
	// Default: "localhost:5432"
	// Environment variable: $BUS_DB_ADDR
	DBAddr string

	// DBUser is the username we use for connecting to postgres.
	// Default: postgres
	// Environment variable: $BUS_DB_USER
	DBUser string

	// RedisAddr is the "host:port" we use for connecting to redis.
	// Default: "localhost:6379"
	// Environment variable: $BUS_REDIS_ADDR
	RedisAddr string

	// TmpDir is the root directory we use for creating temporary
	// files when loading new data.
	// Default: os.TempDir()
	// Environment variable: $BUS_TMP_DIR
	TmpDir string

	// RouteFilter is a pipe-delimited list of route_ids we want
	// to specifically extract from the transit files
	// Default: None (no filter)
	// Environment variable: $BUS_ROUTE_FILTER
	RouteFilter string

	// GTFSURLs is a pipe-delimited list of URLs that have zipped GTFS
	// feeds. https://developers.google.com/transit/gtfs/
	// Default: None
	// Environment variable: $BUS_GTFS_URLS
	GTFSURLs string

	// BusAPIKey is your API key for accessing http://bustime.mta.info/
	// Default: None
	// Environment variable: $MTA_BUS_TIME_API_KEY
	BusAPIKey string

	// SubwayAPIKey is your API key for accessing http://datamine.mta.info/
	// Default: None
	// Environment variable: $MTA_SUBWAY_TIME_API_KEY
	SubwayAPIKey string
)

// ConfigVar reads from the environment variable env or defaults to def. The
// actual value will be written to the string pointed by value. If required is
// true, we'll panic if no non-empty value can be found. If required is false,
// we'll allow empty values.
func ConfigVar(value *string, def, env string, required bool) {
	envValue := os.Getenv(env)

	if len(envValue) > 0 {
		*value = envValue
	} else if len(def) > 0 {
		*value = def
	} else if required {
		log.Panicf("no value for %v", env)
	}

}

// MustDB returns an *sqlx.DB or panics
func MustDB() *sqlx.DB {
	host, port, err := net.SplitHostPort(DBAddr)
	if err != nil {
		log.Panic(err)
	}

	db, err := sqlx.Connect("postgres",
		fmt.Sprintf(
			"user=%s host=%s port=%s sslmode=disable",
			DBUser, host, port,
		),
	)
	if err != nil {
		log.Panic(err)
	}

	return db
}
