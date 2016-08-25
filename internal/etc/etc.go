package etc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/brnstz/bus/internal/conf"
	"github.com/fzzy/radix/redis"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	// redisConnectTimeout is how long we wait to connect to redis
	// before giving up
	redisConnectTimeout = 1 * time.Second
)

var (
	// DBConn is our shared connection to postgres
	DBConn *sqlx.DB

	// httpClient is an http.Client with a reasonable timeout for contacting
	// external sites.
	httpClient = http.Client{Timeout: time.Duration(20) * time.Second}
)

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

// TimeToDepartureSecs takes a time value and converts it to the number
// of seconds since the start of the day
func TimeToDepartureSecs(t time.Time) int {
	return t.Hour()*3600 + t.Minute()*60 + t.Second()
}

// RedisGet retrieves the data cached at k or returns an error if there is no
// cached version or there's another error.
func RedisGet(k string) (b []byte, err error) {
	c, err := redis.DialTimeout("tcp", conf.Cache.RedisAddr, redisConnectTimeout)
	if err != nil {
		log.Println("can't connect to redis", err)
		return
	}

	b, err = c.Cmd("get", k).Bytes()
	if err != nil {
		log.Println("can't get from redis", err)
		return
	}

	return
}

// RedisCache saves bytes to redis using key k.
func RedisCache(k string, b []byte) (err error) {
	c, err := redis.DialTimeout("tcp", conf.Cache.RedisAddr, redisConnectTimeout)
	if err != nil {
		log.Println("can't connect to redis", err)
		return
	}

	// Save the data to redis
	err = c.Cmd("set", k, b, "ex", strconv.Itoa(conf.Cache.RedisTTL)).Err
	if err != nil {
		log.Println("can't set value in redis")
		return
	}

	return
}

// RedisCacheURL takes a URL and returns the bytes of the response from running
// a GET on that URL. Responses are cached for redisTTL seconds. If Redis
// is not available, an error is logged and we hit the URL directly.
func RedisCacheURL(u string) (b []byte, err error) {
	// Get the value from the URL. If we can't do this, it's an error
	// we should return.
	resp, err := httpClient.Get(u)
	if err != nil {
		log.Println("can't get URL", err)
		return
	}
	defer resp.Body.Close()

	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("can't read body", err)
		return
	}

	c, err := redis.DialTimeout("tcp", conf.Cache.RedisAddr, redisConnectTimeout)
	if err != nil {
		log.Println("can't connect to redis", err)
		return
	}

	// Save the data to redis
	err = c.Cmd("set", u, b, "ex", strconv.Itoa(conf.Cache.RedisTTL)).Err
	if err != nil {
		log.Println("can't set value in redis")
		return
	}

	return
}

// MustDB returns an *sqlx.DB or panics
func MustDB() *sqlx.DB {
	host, port, err := net.SplitHostPort(conf.DB.Addr)
	if err != nil {
		log.Panic(err)
	}

	db, err := sqlx.Connect("postgres",
		fmt.Sprintf(
			"user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
			conf.DB.User, conf.DB.Password, host, port, conf.DB.Name,
		),
	)
	if err != nil {
		log.Panic(err)
	}

	db = db.Unsafe()

	return db
}

// BaseTime takes a time and returns the same time with the hour, minute, second
// and nanosecond values set to zero, so that it represents the start
// of the day
func BaseTime(t time.Time) time.Time {
	t = t.Add(-time.Hour * time.Duration(t.Hour()))
	t = t.Add(-time.Minute * time.Duration(t.Minute()))
	t = t.Add(-time.Second * time.Duration(t.Second()))
	t = t.Add(-time.Nanosecond * time.Duration(t.Nanosecond()))

	return t
}

// CreateIntIDs takes  aslice of ints and returns a single sting
// suitable for substitution into an IN clause
func CreateIntIDs(ids []int) string {
	strIDs := make([]string, len(ids))

	for i, _ := range ids {
		strIDs[i] = strconv.Itoa(ids[i])
	}

	return strings.Join(strIDs, ",")
}

// CreateIDs turns a slice of strings into a single string suitable
// for substitution into an IN clause.
func CreateIDs(ids []string) string {
	escapedIDs := make([]string, len(ids))

	// If there are no ids, we want a single blank value
	if len(ids) < 1 {
		return `''`
	}

	for i, _ := range ids {
		escapedIDs[i] = escape(ids[i])
	}

	return strings.Join(escapedIDs, ",")
}

// escape ensures any single quotes inside of id are escaped / quoted
// before creating an ad-hoc string for the IN query
func escape(id string) string {
	var b bytes.Buffer

	b.WriteRune('\u0027')

	for _, char := range id {
		switch char {
		case '\u0027':
			b.WriteRune('\u0027')
			b.WriteRune('\u0027')
		default:
			b.WriteRune(char)
		}
	}

	b.WriteRune('\u0027')

	return b.String()
}

const rad = math.Pi / 180.0
const deg = 180.0 / math.Pi

// stolen from https://github.com/kellydunn/golang-geo/blob/master/point.go
// in turn stolen from http://www.movable-type.co.uk/scripts/latlong.html
func Bearing(lat1, lon1, lat2, lon2 float64) float64 {
	dlon := (lon2 - lon1) * rad
	dlat1 := lat1 * rad
	dlat2 := lat2 * rad

	y := math.Sin(dlon) * math.Cos(dlat2)
	x := math.Cos(dlat1)*math.Sin(dlat2) -
		math.Sin(dlat1)*math.Cos(dlat2)*math.Cos(dlon)

	return math.Atan2(y, x) * deg
}
