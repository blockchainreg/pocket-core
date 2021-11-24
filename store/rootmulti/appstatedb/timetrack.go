package appstatedb

import (
	"fmt"
	"time"
)

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	fmt.Println(fmt.Sprintf("%s took %s", name, elapsed))
}
