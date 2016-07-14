package etc

import (
	"fmt"
	"io/ioutil"
	"log"
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
	redisConnectTimeout = 100 * time.Millisecond
)

var (
	// DBConn is our shared connection to postgres
	DBConn *sqlx.DB
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

// RedisGet retrieves a cached version of the incoming URL or returns an
// error if there is no cached version or there's another error.
func RedisGet(u string) (b []byte, err error) {
	c, err := redis.DialTimeout("tcp", conf.Cache.RedisAddr, redisConnectTimeout)
	if err != nil {
		log.Println("can't connect to redis", err)
		return
	}

	b, err = c.Cmd("get", u).Bytes()
	if err != nil {
		log.Println("can't get from redis", err)
		return
	}

	return
}

// RedisCache takes a URL and returns the bytes of the response from running a
// GET on that URL. Responses are cached for redisTTL seconds. If Redis
// is not available, an error is logged and we hit the URL directly.
func RedisCache(u string) (b []byte, err error) {
	// Get the value from the URL. If we can't do this, it's an error
	// we should return.
	resp, err := http.Get(u)
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
		// Log redis errors and then ignore. We may still be able to get
		// our data even without redis.
		log.Println("can't connect to redis", err)
		return
	}

	// Save the data to redis
	err = c.Cmd("set", u, b, "ex", strconv.Itoa(conf.Cache.RedisTTL)).Err
	if err != nil {
		// Log redis errors and then ignore. We still have our bytes
		// that we can return, so it's not an error for the client.
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
			"user=%s host=%s port=%s dbname=%s sslmode=disable",
			conf.DB.User, host, port, conf.DB.Name,
		),
	)
	if err != nil {
		log.Panic(err)
	}

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
