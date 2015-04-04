package common

import (
	"io/ioutil"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fzzy/radix/redis"
	"github.com/jmoiron/sqlx"

	"fmt"
	"os"

	_ "github.com/lib/pq"
)

var (
	DB = mustDB()

	metersInMile = 1609.344

	redisTTL = 30

	SecsAfterMidnight = 60 * 60 * 24

	cpu = mustCPU()
)

func mustCPU() bool {
	nc := runtime.NumCPU()
	log.Println("setting go procs to cpu count: ", nc)

	runtime.GOMAXPROCS(nc)

	return true
}

func mustDB() *sqlx.DB {
	ip := os.Getenv("BUS_DB_HOST")

	db, err := sqlx.Connect("postgres",
		fmt.Sprintf(
			"user=postgres host=%v sslmode=disable",
			ip,
		),
	)
	if err != nil {
		panic(err)
	}

	return db
}

func MileToMeter(miles float64) float64 {
	return miles * metersInMile
}

func MeterToMile(meters float64) float64 {
	return meters / metersInMile
}

func TimeStrToSecs(timeStr string) int {
	parts := strings.Split(timeStr, ":")

	hr, err := strconv.Atoi(parts[0])
	if err != nil {
		panic(err)
	}

	min, err := strconv.Atoi(parts[1])
	if err != nil {
		panic(err)
	}

	sec, err := strconv.Atoi(parts[2])
	if err != nil {
		panic(err)
	}

	return sec + min*60 + hr*3600
}

func SecsToTimeStr(secs int) string {
	hr := secs / 3600

	secs = secs % 3600

	min := secs / 60

	secs = secs % 60

	return fmt.Sprintf("%02d:%02d:%02d", hr, min, secs)
}

// Check in redis cache for URL, otherwise get and set it
func RedisCache(u string) (b []byte, err error) {
	c, err := redis.DialTimeout(
		"tcp", fmt.Sprintf("%v:6379",
			os.Getenv("BUS_REDIS_HOST")), time.Second*1,
	)
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
