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
	// redisTTL is how many seconds we cache things in redis
	redisTTL = 30

	// redisConnectTimeout is how long we wait to connect to redis
	// before giving up
	redisConnectTimeout = 1 * time.Second
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

// RedisCache takes a URL and returns the bytes of the response from running a
// GET on that URL. Responses are cached for redisTTL seconds. If Redis
// is not available, an error is logged and we hit the URL directly.
func RedisCache(u string) (b []byte, err error) {
	c, err := redis.DialTimeout("tcp", conf.API.RedisAddr, redisConnectTimeout)
	if err != nil {
		// Log redis errors and then ignore. We may still be able to get
		// our data even without redis.
		log.Println("can't connect to redis", err)
		err = nil
	}

	// If we have a redis connection, try to get response there first. If we
	// succeed, return early.
	if c != nil {
		b, err = c.Cmd("get", u).Bytes()
		if err == nil {
			return
		}
	}

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

	// If we have a redis connection, save the value.
	if c != nil {
		err = c.Cmd("set", u, b, "ex", strconv.Itoa(redisTTL)).Err

		if err != nil {
			// Log redis errors and then ignore. We still have our bytes
			// that we can return, so it's not an error for the client.
			log.Println("can't set value in redis")
			err = nil
		}
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
