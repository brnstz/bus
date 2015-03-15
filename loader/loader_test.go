package loader_test

import (
	"fmt"
	"testing"

	"github.com/brnstz/bus/loader"
)

func TestLoader(t *testing.T) {
	l := loader.NewLoader("../schema/brooklyn/")

	/*
		for i, v := range l.Stops {
			fmt.Println(i, v)
		}
	*/

	for i, v := range l.ScheduledStopTimes {
		fmt.Println(i, v)
	}
}
