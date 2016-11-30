package gover

import (
	"context"
	"fmt"
	"time"
)

//containers for all gotermins
//can be used as main controller or to retain informations
type CrontabMinE struct {
	//list of registered gotermins in form of a map
	//the map key is the register name
	cronjobs map[string]*Gotermin
	//the set timezone
	//all gotermins will run in this timezone
	timeLocation *time.Location
}

//create new container with a certain time location
//return error if time location is empty
func NewCrontab(loc *time.Location) (*CrontabMinE, error) {
	if loc == nil {
		return nil, fmt.Errorf("Please input a valid time location")
	}

	return &CrontabMinE{
		cronjobs:     map[string]*Gotermin{},
		timeLocation: loc,
	}, nil
}

//register gotermins on the crontab with key
//the requirement is exactly the same for each category
//only this time use location from crontab
//return error if failed to create the gotermin
func (ct *CrontabMinE) RegisterNewHourly(key string, job func(context.Context), minute string) error {
	return ct.registerNew("hourly", key, job, minute)
}

func (ct *CrontabMinE) RegisterNewDaily(key string, job func(context.Context), hour string) error {
	return ct.registerNew("daily", key, job, hour)
}

func (ct *CrontabMinE) RegisterNewWeekly(key string, job func(context.Context), weekly string) error {
	return ct.registerNew("weekly", key, job, weekly)
}

func (ct *CrontabMinE) RegisterNewCustomInterval(key string, job func(context.Context), customInterval time.Duration) error {
	return ct.registerNew("custom", key, job, customInterval)
}

func (ct *CrontabMinE) registerNew(cat, key string, job func(context.Context), input interface{}) error {
	//return error if duplicate key is found
	if _, ok := ct.cronjobs[key]; ok {
		return fmt.Errorf("Duplicate key for %s is found", key)
	}

	//create new gotermin, return error if failed to
	//switch the category, also return return error if category is invalid
	var gotermin *Gotermin
	var err error

	switch cat {
	case "hourly":
		if minute, ok := input.(string); !ok {
			return fmt.Errorf("Invalid input type")
		} else {
			gotermin, err = NewHourly(job, minute, ct.timeLocation)

		}
	case "daily":
		if hour, ok := input.(string); !ok {
			return fmt.Errorf("Invalid input type")
		} else {
			gotermin, err = NewDaily(job, hour, ct.timeLocation)
		}
	case "weekly":
		if weekly, ok := input.(string); !ok {
			return fmt.Errorf("Invalid input type")
		} else {
			gotermin, err = NewWeekly(job, weekly, ct.timeLocation)
		}
	case "custom":
		if customInterval, ok := input.(time.Duration); !ok {
			return fmt.Errorf("Invalid input type")
		} else {
			gotermin, err = NewCustomInterval(job, customInterval, ct.timeLocation)
		}
	default:
		return fmt.Errorf("Invalid category")
	}

	if err != nil {
		return err
	}

	//if there's no error then add the key into crontab
	ct.cronjobs[key] = gotermin
	return nil
}

//start all inactive gotermins
//return error if any of them is failing
func (ct *CrontabMinE) StartAll() error {
	for _, gotermin := range ct.cronjobs {
		if !gotermin.isActive {
			if err := gotermin.Start(); err != nil {
				return err
			}
		}
	}
	return nil
}

//start a certain gotermin
//return error if key is not found or failed to start
func (ct *CrontabMinE) Start(key string) error {
	gotermin, found := ct.cronjobs[key]
	if !found {
		return fmt.Errorf("Key %s is not found", key)
	}

	return gotermin.Start()
}

//stop all active gotermins
//return error if any of them is failing
func (ct *CrontabMinE) StopAll() {
	for _, gotermin := range ct.cronjobs {
		go gotermin.Stop()
	}
}

//stop a certain gotermin
//return error if key is not found or failed to stop
func (ct *CrontabMinE) Stop(key string) error {
	gotermin, found := ct.cronjobs[key]
	if !found {
		return fmt.Errorf("Key %s is not found", key)
	}

	return gotermin.Stop()
}

//return the summary of current crontab
func (ct CrontabMinE) String() string {
	result := fmt.Sprintf(`
Summary
Key-----Interval-----StartingPoint-----Status`)

	for key, cronjob := range ct.cronjobs {
		interval := cronjob.intervalCategory
		if interval == "custom" {
			interval += fmt.Sprintf(" (%+v)", cronjob.customInterval)
		}

		isActive := "inactive"
		if cronjob.isActive {
			isActive = "active"
		}

		startingPoint := cronjob.startingPoint
		if startingPoint == "" {
			startingPoint = "immediately"
		}

		result += fmt.Sprintf(`
%s-----%s-----%s-----%s`, key, interval, startingPoint, isActive)
	}

	return result
}
