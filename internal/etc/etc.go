package etc

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/brnstz/bus/internal/conf"
	"github.com/fzzy/radix/redis"
	_ "github.com/lib/pq"
)

const (
	// metersInMile is a constant for converting between meters and miles
	metersInMile = 1609.344
)

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
// GET on that URL. Responses are cached for conf.RedisTTL seconds.
func RedisCache(u string) (b []byte, err error) {
	c, err := redis.DialTimeout("tcp", conf.RedisAddr, conf.RedisConnectTimeout)
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

	err = c.Cmd("set", u, b, "ex", strconv.Itoa(conf.RedisTTL)).Err

	if err != nil {
		log.Println("can't set value in redis")
		return
	}

	return
}
