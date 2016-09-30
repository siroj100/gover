//GoTermin is in principal a cronjob like scheduler
//it will do its assigned on a certain schedule
//the job function in this case is a simple func(context.Context) without returning anything
package gover

import (
	"context"
	"fmt"
	"math"
	"strconv"
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

func (ct *CrontabMinE) RegisterNewCustomeInterval(key string, job func(context.Context), customInterval time.Duration) error {
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

type Gotermin struct {
	//the job that's supposed to be done
	//it will run on separate thread
	//the job has context as input so it can handle the timeout from each interval
	Job func(ctx context.Context)
	//channel to stop the job
	quit chan interface{}
	//interval to decide the context timeout
	intervalCategory string
	//state when the first interval should be started
	//it varies depending on category
	//however it should be validated for each initiation
	startingPoint string
	//indicator whether it's still running or not
	isActive bool
	//the time location
	timeLocation *time.Location
	//container for custom interval
	customInterval time.Duration
}

//this should setup a gotermin, which will run in 1 hour interval
//input minute decide when (minute) the schedule should be started (number between 00-60)
//if input minute is an empty string, start the job immediately
//also determine the time location to make sure it's running properly
func NewHourly(job func(context.Context), minute string, loc *time.Location) (*Gotermin, error) {
	//return error if minute is not a valid minute string
	//add exception for empty string
	if _, err := time.Parse("04", minute); err != nil && minute != "" {
		return nil, fmt.Errorf("Please input minute between 00-59", minute)
	}

	//also return error if time location is nil
	if loc == nil {
		return nil, fmt.Errorf("Please input a valid time location")
	}

	return &Gotermin{
		Job:              job,
		quit:             make(chan interface{}, 1),
		intervalCategory: "hourly",
		startingPoint:    minute,
		timeLocation:     loc,
	}, nil
}

//this function will schedule the job in daily interval
//input hour will decide when the schedule should be started
//the hour should be in form hhmm, if it's not parseable then return error
//also determine the time location to make sure it's running properly
func NewDaily(job func(context.Context), hour string, loc *time.Location) (*Gotermin, error) {
	//return error if hour is not a valid hour string
	//add exception for empty string (the schedule will run immediately)
	if _, err := time.Parse("1504", hour); err != nil && hour != "" {
		return nil, fmt.Errorf("Please input correct time in format hhmm")
	}

	//also return error if time location is nil
	if loc == nil {
		return nil, fmt.Errorf("Please input a valid time location")
	}

	return &Gotermin{
		Job:              job,
		quit:             make(chan interface{}, 1),
		intervalCategory: "daily",
		startingPoint:    hour,
		timeLocation:     loc,
	}, nil
}

//this function will set the schedule interval at will
//however the starting point can't be set (i.e. the job will start immediately)
//and the custom interval can't be less than 1 second
func NewCustomInterval(job func(context.Context), interval time.Duration, loc *time.Location) (*Gotermin, error) {
	//return error if location is nil
	if loc == nil {
		return nil, fmt.Errorf("Please input a valid time location")
	}

	//it should not less than 1 second
	if interval.Seconds() < float64(1) {
		return nil, fmt.Errorf("Please insert duration greater than 1 second")
	}

	//set the interval category into custom
	//set the custom interval into desired interval
	return &Gotermin{
		Job:              job,
		quit:             make(chan interface{}, 1),
		intervalCategory: "custom",
		startingPoint:    "",
		timeLocation:     loc,
		customInterval:   interval,
	}, nil

}

//stop the currently running go termin
func (gt *Gotermin) Stop() error {
	if !gt.isActive {
		return fmt.Errorf("The scheduler is already inactive")
	}
	gt.quit <- "stop"
	return nil
}

//run the scheduler
//validate etc before starting the loop
func (gt *Gotermin) Start() error {
	//validate the entry again
	//return error if it's still active
	if gt.isActive {
		return fmt.Errorf("The scheduler is still active currently")
	}

	//check the interval category
	//return error if it's not valid
	//also check the respective starting point altogether
	//determine the timeout interval according to category as well
	var jobInterval time.Duration
	switch gt.intervalCategory {
	case "hourly":
		//set the interval into 1 hour
		jobInterval = time.Hour
	case "daily":
		//set the interval into 24 hours
		jobInterval = time.Hour * 24
	case "custom":
		//set the job interval according to custom interval
		jobInterval = gt.customInterval
	default:
		return fmt.Errorf("Current category is invalid: %s", gt.intervalCategory)
	}

	//now decide how long we should wait until the starting point
	//use time sleep for this action
	//determine the sleep duration from category
	sleepDuration, err := gt.durationUntilFirst()
	if err != nil {
		return err
	}

	//if there is nothing wrong then start the job
	go gt.start(jobInterval, sleepDuration)

	return nil
}

func (gt *Gotermin) start(jobInterval, sleepDuration time.Duration) {
	//first of all set the status into running
	gt.isActive = true

	//then sleep for the assigned sleep duration
	time.Sleep(sleepDuration)

	//after awoken from the slumber
	//start an infinite loop with the job interval
	for {
		//create new context to make sure the job interval works as planned
		ctx, cancel := context.WithTimeout(context.Background(), jobInterval)

		//then simply do the job in different thread
		go gt.Job(ctx)

		//wait until either context is timed out or it's stopped
		select {
		case <-ctx.Done():
			continue
		case signal := <-gt.quit:
			//if the quite channel is filled, stopping the loop
			//also cancel the context and set status into inactive
			fmt.Println("Stopping jobs with signal: ", signal)
			cancel()
			gt.isActive = false
			return
		}

	}
}

//return how long we should wait until the first action should be done
//also check the category, return error if not valid
func (gt *Gotermin) durationUntilFirst() (time.Duration, error) {
	var result time.Duration

	//add exception for empty string
	//that means that we do not need to wait, just do it immediately
	if gt.startingPoint == "" {
		return result, nil
	}

	//load the current regional time
	timeNow := time.Now().In(gt.timeLocation)
	switch gt.intervalCategory {
	case "hourly":
		//check whether starting point is minute parseable
		if _, err := time.Parse("04", gt.startingPoint); err != nil {
			return result, fmt.Errorf("Invalid starting point for hourly schedule")
		}

		//this means that we have to wait for 0-59 minutes
		//to be precise, parse the value in seconds
		currentMinutes, currentSeconds := timeNow.Format("04"), timeNow.Format("05")
		//assume that parse float64 should be successful
		curMin, _ := strconv.ParseFloat(currentMinutes, 64)
		curSec, _ := strconv.ParseFloat(currentSeconds, 64)
		//calculate the total seconds passed
		totalSec := curMin*60 + curSec

		//also parse the starting point
		//return error if failed
		mins, err := strconv.ParseFloat(gt.startingPoint, 64)
		if err != nil {
			return result, err
		}

		//now we can calculate the duration in seconds
		durFloat := math.Abs(totalSec - mins*60)

		//parse the duration into time.Duration
		result, err = time.ParseDuration(fmt.Sprintf("%.0fs", durFloat))
	case "daily":
		//check whether starting point is into hour parseable
		if _, err := time.Parse("1504", gt.startingPoint); err != nil {
			return result, fmt.Errorf("Invalid starting point for daily schedule")
		}

		//return how many seconds we have to wait
		currentHours, currentMinutes, currentSeconds := timeNow.Format("15"), timeNow.Format("04"), timeNow.Format("05")
		curHour, _ := strconv.ParseFloat(currentHours, 64)
		curMin, _ := strconv.ParseFloat(currentMinutes, 64)
		curSec, _ := strconv.ParseFloat(currentSeconds, 64)
		//calculate the total seconds passed
		totalSec := curHour*60*60 + curMin*60 + curSec

		//parse the starting point
		//return error if failed
		timeDur, err := time.ParseDuration(fmt.Sprintf("%sh%sm", gt.startingPoint[:2], gt.startingPoint[2:]))
		if err != nil {
			return result, err
		}

		//calculate the wait duration in seconds
		durFloat := math.Abs(totalSec - timeDur.Seconds())

		//parse the duration into time.Duration
		result, err = time.ParseDuration(fmt.Sprintf("%.0fs", durFloat))
	default:
		//return error if category is not valid
		return result, fmt.Errorf("Current category is invalid: %s", gt.intervalCategory)
	}

	return result, nil
}
