package common

import (
	"log"
	"runtime"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"

	"fmt"
	"os"

	_ "github.com/lib/pq"
)

var (
	DB = mustDB()

	metersInMile = 1609.344

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
