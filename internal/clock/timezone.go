package clock

import (
	"fmt"
	"time"
)

const ChinaTimezoneName = "Asia/Shanghai"

func ChinaLocation() (*time.Location, error) {
	location, err := time.LoadLocation(ChinaTimezoneName)
	if err != nil {
		return nil, fmt.Errorf("load timezone %s: %w", ChinaTimezoneName, err)
	}

	return location, nil
}

func SetSystemLocationToChina() error {
	location, err := ChinaLocation()
	if err != nil {
		return err
	}

	time.Local = location
	return nil
}
