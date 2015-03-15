package loader_test

import (
	"fmt"
	"testing"

	"github.com/brnstz/bus/loader"
)

func TestLoader(t *testing.T) {
	l := loader.NewLoader("../schema/subway/")

	for i, v := range l.Stops {
		fmt.Println(i, v)
	}
}
