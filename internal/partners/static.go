package partners

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/brnstz/bus/internal/etc"
	"github.com/brnstz/bus/internal/models"
)

// static is a default partner that reads vehicle locations from the database
type static struct{}

func (p static) Precache(agencyID, routeID string, directionID int) error {
	k := fmt.Sprintf("%v|%v|%v", agencyID, routeID, directionID)
	now := time.Now()

	today := etc.BaseTime(now)
	todayName := strings.ToLower(now.Format("Monday"))
	todayIDs, err := models.GetNewServiceIDs(etc.DBConn, agencyID, todayName, today)
	if err != nil {
		log.Println("can't get serviceIDs", err)
		return err
	}

	vehicles, err := models.GetVehicles(
		agencyID, routeID, directionID, todayIDs, etc.TimeToDepartureSecs(now),
	)

	b, err := json.Marshal(vehicles)
	if err != nil {
		log.Println("can't marshal vehicles", err)
		return err
	}

	err = etc.RedisCache(k, b)
	if err != nil {
		log.Println("can't save static vehicles to redis", err)
		return err
	}

	log.Println("succesfully saved", k)

	return nil
}

func (p static) Live(agencyID, routeID, stopID string, directionID int) (d models.Departures, v []models.Vehicle, err error) {
	k := fmt.Sprintf("%v|%v|%v", agencyID, routeID, directionID)

	b, err := etc.RedisGet(k)
	if err != nil {
		log.Println("can't get from redis", err)
		return
	}

	err = json.Unmarshal(b, &v)
	if err != nil {
		log.Println("can't unmarshal cached vehicles", err)
		return
	}

	// we have no departures to append, ensure it's blank
	d = make(models.Departures, 0)

	return
}
