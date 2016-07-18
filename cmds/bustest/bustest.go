package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	_ "net/http/pprof"

	"github.com/brnstz/bus/internal/conf"
	"github.com/brnstz/bus/internal/etc"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	var err error
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	err = envconfig.Process("bus", &conf.DB)
	if err != nil {
		log.Fatal(err)
	}

	err = envconfig.Process("bus", &conf.Loader)
	if err != nil {
		log.Fatal(err)
	}

	time.Local, err = time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(err)
	}

	/*
		next := map[string]string{
			"Friday":    "Saturday",
			"Saturday":  "Sunday",
			"Sunday":    "Monday",
			"Monday":    "Tuesday",
			"Tuesday":   "Wednesday",
			"Wednesday": "Thursday",
			"Thursday":  "Friday",
		}
	*/

	prev := map[string]string{
		"Friday":    "Thursday",
		"Saturday":  "Friday",
		"Sunday":    "Saturday",
		"Monday":    "Sunday",
		"Tuesday":   "Monday",
		"Wednesday": "Tuesday",
		"Thursday":  "Wednesday",
	}

	day := etc.BaseTime(time.Now())
	last := "Monday"
	for i := 0; i < 5000; i++ {
		name := day.Format("Monday")

		fmt.Printf("last: %v, next: %v\n", last, name)
		if prev[last] != name {
			log.Fatal(day)
		}

		last = name

		day = day.AddDate(0, 0, -1)
	}
}
