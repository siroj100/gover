//GoTermin is in principal a cronjob like scheduler
//it will do its assigned on a certain schedule
//the job function in this case is a simple func(context.Context) without returning anything
package gover

import (
	"context"
	"fmt"
	"strings"
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
	//this interface contain the interval and sleep duration
	jobInterval interval
	//indicator whether it's still running or not
	isActive bool
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
		Job:         job,
		quit:        make(chan interface{}, 1),
		jobInterval: hourlyJob{minute, loc},
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
		Job:         job,
		quit:        make(chan interface{}, 1),
		jobInterval: dailyJob{hour, loc},
	}, nil
}

//this function will schedule the job in weekly interval
//input weekday is in format of "weekday hour" separated by @ symbol (e.g. "Monday@1530")
//if input is not valid then an error will be returned
func NewWeekly(job func(context.Context), weekly string, loc *time.Location) (*Gotermin, error) {
	//weekly string should contains exactly 2 elements after splitted by @
	weeklySplitted := strings.Split(weekly, "@")
	if len(weeklySplitted) != 2 {
		return nil, fmt.Errorf("Please input correct weekly string in format Weekday@hhmm")
	}

	//validate the week first
	if _, err := getWeekDuration(weeklySplitted[0]); err != nil {
		return nil, err
	}

	//and then the hour
	if _, err := time.Parse("1504", weeklySplitted[1]); err != nil {
		return nil, err
	}

	//also return error if time location is nil
	if loc == nil {
		return nil, fmt.Errorf("Please input a valid time location")
	}

	return &Gotermin{
		Job:         job,
		quit:        make(chan interface{}, 1),
		jobInterval: weeklyJob{weekly, loc},
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
		Job:         job,
		quit:        make(chan interface{}, 1),
		jobInterval: customIntervalJob{interval},
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
	jobInterval := gt.jobInterval.getInterval()

	//now decide how long we should wait until the starting point
	//use time sleep for this action
	//determine the sleep duration from category
	sleepDuration, err := gt.jobInterval.getSleepDuration(time.Now())
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
