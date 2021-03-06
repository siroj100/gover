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
		return nil, TimeLocationError
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
	//return error if duplicate key is found
	if _, ok := ct.cronjobs[key]; ok {
		return DuplicateKeyError
	}

	if gotermin, err := NewHourly(job, minute, ct.timeLocation); err != nil {
		return err
	} else {
		//if there's no error then add the key into crontab
		ct.cronjobs[key] = gotermin
	}

	return nil
}

func (ct *CrontabMinE) RegisterNewDaily(key string, job func(context.Context), hour string) error {
	//return error if duplicate key is found
	if _, ok := ct.cronjobs[key]; ok {
		return DuplicateKeyError
	}

	if gotermin, err := NewDaily(job, hour, ct.timeLocation); err != nil {
		return err
	} else {
		//if there's no error then add the key into crontab
		ct.cronjobs[key] = gotermin
	}

	return nil
}

func (ct *CrontabMinE) RegisterNewWeekly(key string, job func(context.Context), weekly string) error {
	//return error if duplicate key is found
	if _, ok := ct.cronjobs[key]; ok {
		return DuplicateKeyError
	}

	if gotermin, err := NewWeekly(job, weekly, ct.timeLocation); err != nil {
		return err
	} else {
		//if there's no error then add the key into crontab
		ct.cronjobs[key] = gotermin
	}

	return nil
}

func (ct *CrontabMinE) RegisterNewCustomInterval(key string, job func(context.Context), customInterval time.Duration) error {
	//return error if duplicate key is found
	if _, ok := ct.cronjobs[key]; ok {
		return DuplicateKeyError
	}

	if gotermin, err := NewCustomInterval(job, customInterval, ct.timeLocation); err != nil {
		return err
	} else {
		//if there's no error then add the key into crontab
		ct.cronjobs[key] = gotermin
	}

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
		return KeyNotFoundError
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
		return KeyNotFoundError
	}

	return gotermin.Stop()
}

//return the summary of current crontab
func (ct CrontabMinE) String() string {
	result := fmt.Sprintf(`
Summary
Key-----[Interval] StartingPoint-----Status`)

	for key, cronjob := range ct.cronjobs {
		isActive := "inactive"
		if cronjob.isActive {
			isActive = "active"
		}

		result += fmt.Sprintf(`
%s-----%s-----%s`, key, cronjob.jobInterval, isActive)
	}

	return result
}

//get all active keys from a crontab struct
func (ct CrontabMinE) GetAllKeys() []string {
	var result []string
	for key, _ := range ct.cronjobs {
		result = append(result, key)
	}
	return result
}

//get only active keys from a crontab struct
func (ct CrontabMinE) GetActiveKeys() []string {
	var result []string
	for key, val := range ct.cronjobs {
		if val.isActive {
			result = append(result, key)
		}
	}
	return result
}

//get only inactive keys from a crontab struct
func (ct CrontabMinE) GetInactiveKeys() []string {
	var result []string
	for key, val := range ct.cronjobs {
		if !val.isActive {
			result = append(result, key)
		}
	}
	return result
}

//get a GoTermin by a key
//return error if not found
func (ct CrontabMinE) GetCronjob(key string) (*Gotermin, error) {
	if gt, ok := ct.cronjobs[key]; ok {
		return gt, nil
	}
	return nil, KeyNotFoundError
}
