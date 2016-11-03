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
	wakeUp := time.After(sleepDuration)

	//return if signal quit is received
	select {
	case <-gt.quit:
		//return and set the status into inactive
		gt.isActive = false
		return
	case <-wakeUp:
		//continue to start the job periodically
	}

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
	var err error

	//add exception for empty string
	//that means that we do not need to wait, just do it immediately
	if gt.startingPoint == "" {
		return result, nil
	}

	//load the current regional time
	timeNow := time.Now().In(gt.timeLocation)
	switch gt.intervalCategory {
	case "hourly":
		if result, err = gt.calculateHourlyDuration(timeNow); err != nil {
			return result, err
		}
	case "daily":
		if result, err = gt.calculateDailyDuration(timeNow); err != nil {
			return result, err
		}
	default:
		//return error if category is not valid
		return result, fmt.Errorf("Current category is invalid: %s", gt.intervalCategory)
	}

	return result, nil
}

//functions to calculate the duration for each category
func (gt *Gotermin) calculateHourlyDuration(timeNow time.Time) (time.Duration, error) {
	var result time.Duration

	//check whether starting point is minute parseable
	if _, err := time.Parse("04", gt.startingPoint); err != nil {
		return result, fmt.Errorf("Invalid starting point for hourly schedule")
	}

	//add time difference based on location
	timeNow = timeNow.In(gt.timeLocation).Add(calculateTimeDiff(gt.timeLocation))

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
	//add an hour if the startingSec is less than totalSec
	startingSec := mins * 60
	if startingSec < totalSec {
		startingSec += 60 * 60
	}

	durFloat := math.Abs(startingSec - totalSec)

	//parse the duration into time.Duration
	return time.ParseDuration(fmt.Sprintf("%.0fs", durFloat))
}

func (gt *Gotermin) calculateDailyDuration(timeNow time.Time) (time.Duration, error) {
	var result time.Duration

	//check whether starting point is into hour parseable
	if _, err := time.Parse("1504", gt.startingPoint); err != nil {
		return result, fmt.Errorf("Invalid starting point for daily schedule")
	}

	timeThen, _ := time.Parse("20060102", timeNow.Format("20060102"))
	dur, _ := time.ParseDuration(fmt.Sprintf("%sh%sm", gt.startingPoint[:2], gt.startingPoint[2:]))
	//add the time difference between server and the selected time location
	timeThen = timeThen.Add(dur).In(gt.timeLocation).Add(calculateTimeDiff(gt.timeLocation))

	//add 1 day to timeThen if it's before time now
	if timeNow.After(timeThen) {
		timeThen = timeThen.AddDate(0, 0, 1)
	}

	return timeThen.Sub(timeNow), nil
}

//calculate time difference between selected time location and server local time
//for example if the server is run in UTC and the selected time is GMT (UTC+7)
//then this function should yield 7hours
func calculateTimeDiff(loc *time.Location) time.Duration {
	serverTime := time.Now().Format("-0700")
	localTime := time.Now().In(loc).Format("-0700")

	//parse the time zone duration of server and local time
	serverDur, _ := time.ParseDuration(fmt.Sprintf("%sh%sm", serverTime[1:3], serverTime[3:5]))
	localDur, _ := time.ParseDuration(fmt.Sprintf("%sh%sm", localTime[1:3], localTime[3:5]))

	//convert both durations into seconds float
	//however if the sign is "-" multiply it with minus 1
	serverSec := serverDur.Seconds()
	if serverTime[:1] == "-" {
		serverSec = serverSec * -1
	}
	localSec := localDur.Seconds()
	if localTime[:1] == "-" {
		localSec = localSec * -1
	}

	//parse the difference between server and local time
	result, _ := time.ParseDuration(fmt.Sprintf("%.0fs", serverSec-localSec))

	return result
}
