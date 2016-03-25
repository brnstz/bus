package common

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fzzy/radix/redis"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	// metersInMile is a constant for converting between meters and miles
	metersInMile = 1609.344

	// redisTTL is how many seconds we cache things in redis
	redisTTL = 30

	// redisConnectTimeout is how long we wait to connect to redis
	// before giving up
	redisConnectTimeout = 1 * time.Second
)

var (
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

	// BusAPIKey is your API key for accessing http://bustime.mta.info/
	// Default: None
	// Environment variable: $MTA_BUS_TIME_API_KEY
	BusAPIKey string

	// SubwayAPIKey is your API key for accessing http://datamine.mta.info/
	// Default: None
	// Environment variable: $MTA_SUBWAY_TIME_API_KEY
	SubwayAPIKey string
)

// configVar is a struct that helps us initialize variables that have a
// default value that can be overridden by an environment variable.
type configVar struct {
	// value is a pointer to the variable we're setting
	value *string

	// def is the default value is nothing is found in the environment
	def string

	// env is the environment variable name to use
	env string
}

// initialize sets the value pointed to by cv.value.  If there is a value in
// the environment, that value is used. Otherwise, the cv.def is used. If
// neither are non-empty, we panic.
func (cv *configVar) initialize() {
	envValue := os.Getenv(cv.env)

	if len(envValue) > 0 {
		*cv.value = envValue
	} else if len(cv.def) > 0 {
		*cv.value = cv.def
	} else {
		log.Panicf("no value for %v", cv.env)
	}
}

func init() {
	log.SetFlags(log.Lshortfile)

	vars := []configVar{
		{&APIAddr, ":8000", "BUS_API_ADDR"},
		{&DBAddr, "localhost:5432", "BUS_DB_ADDR"},
		{&DBUser, "postgres", "BUS_DB_USER"},
		{&RedisAddr, "localhost:6379", "BUS_REDIS_ADDR"},
		{&TmpDir, os.TempDir(), "BUS_TMP_DIR"},
		{&BusAPIKey, "", "MTA_BUS_TIME_API_KEY"},
		{&SubwayAPIKey, "", "MTA_SUBWAY_TIME_API_KEY"},
	}

	for _, v := range vars {
		v.initialize()
	}

	DB = mustDB()
}

// mustDB returns an *sqlx.DB or panics
func mustDB() *sqlx.DB {
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

// MileToMeter converts miles to meters
func MileToMeter(miles float64) float64 {
	return miles * metersInMile
}

// MeterToMile converts meters to miles
func MeterToMile(meters float64) float64 {
	return meters / metersInMile
}

// TimeStrToSecs takes a string like "01:23:45" (hours:minutes:seconds)
// and returns a integer of the total number of seconds
func TimeStrToSecs(timeStr string) int {
	parts := strings.Split(timeStr, ":")

	hr, err := strconv.Atoi(parts[0])
	if err != nil {
		log.Panic(err)
	}

	min, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Panic(err)
	}

	sec, err := strconv.Atoi(parts[2])
	if err != nil {
		log.Panic(err)
	}

	return sec + min*60 + hr*3600
}

// SecsToTimeStr takes an integer number of seconds and returns a string
// like "01:23:45" (hours:minutes:seconds)
func SecsToTimeStr(secs int) string {
	hr := secs / 3600

	secs = secs % 3600

	min := secs / 60

	secs = secs % 60

	return fmt.Sprintf("%02d:%02d:%02d", hr, min, secs)
}

// RedisCache takes a URL and returns the bytes of the response from running a
// GET on that URL. Responses are cached for redisTTL seconds.
func RedisCache(u string) (b []byte, err error) {
	c, err := redis.DialTimeout("tcp", RedisAddr, redisTTL)
	if err != nil {
		log.Println("can't connect to redis", err)
		return
	}

	b, err = c.Cmd("get", u).Bytes()
	if err == nil {
		log.Printf("found %v in redis cache\n", u)
		return
	}

	log.Println("didn't find value in redis, going to get: ", u)

	resp, err := http.Get(u)
	if err != nil {
		log.Println("can't get URL", err, u)
		return
	}
	defer resp.Body.Close()

	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("can't read body", err, u)
		return
	}

	err = c.Cmd("set", u, b, "ex", strconv.Itoa(redisTTL)).Err

	if err != nil {
		log.Println("can't set value in redis")
		return
	}

	return
}
