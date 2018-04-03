//interval category interface to be used by GoTermin
package gover

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type (
	interval interface {
		//interval duration between jobs
		getInterval() time.Duration
		//sleep duration is time delay before executing the first job
		//input is the to be subtracted time (use time now)
		getSleepDuration(time.Time) (time.Duration, error)
	}
)

///////////////////////////////
//////CUSTOM INTERVAL ////////
/////////////////////////////

//goTermin that runs with custom interval
//this doesn't have any starting point so it will run immediately
type customIntervalJob struct {
	timeInterval time.Duration
}

func (cij customIntervalJob) getInterval() time.Duration { return cij.timeInterval }
func (cij customIntervalJob) getSleepDuration(startTime time.Time) (time.Duration, error) {
	return time.Second * 0, nil
}

func (cij customIntervalJob) String() string {
	return fmt.Sprintf("[%s] immediately", cij.getInterval())
}

///////////////////////////////
////////// HOURLY ////////////
/////////////////////////////

//goTermin that runs in hourly basis
//the interval is simply == 3600 seconds
//the starting point should be in minute parseable
//e.g. "04" or "30", basically from "00" until "59"
type hourlyJob struct {
	startingPoint string
	timeLocation  *time.Location
}

func (hj hourlyJob) getInterval() time.Duration { return time.Hour }

//calculate the sleep duration for hourly category
//the time difference should be accounted on this
func (hj hourlyJob) getSleepDuration(startTime time.Time) (time.Duration, error) {
	var result time.Duration

	//return if starting point is empty string
	if hj.startingPoint == "" {
		return result, nil
	}

	//check whether the starting point is into minute parseable
	if _, err := time.Parse("04", hj.startingPoint); err != nil {
		return result, StartingPointError
	}

	//save the start time on the location
	//make sure it's not nil
	if hj.timeLocation == nil {
		return result, TimeLocationError
	}
	startTime = startTime.In(hj.timeLocation)
	//this means that we have to wait for 0-59 minutes
	//to be precise, parse the value in seconds
	currentMinutes, currentSeconds := startTime.Format("04"), startTime.Format("05")
	//assume that parse float64 should be successful
	curMin, _ := strconv.ParseFloat(currentMinutes, 64)
	curSec, _ := strconv.ParseFloat(currentSeconds, 64)

	//calculate the total seconds passed
	totalSec := curMin*60 + curSec

	//also parse the starting point into float
	//no need to handle error since it's already validated previously
	mins, _ := strconv.ParseFloat(hj.startingPoint, 64)

	//now we can calculate the duration in seconds
	//add an hour if the startingSec is less than totalSec
	//also add the time difference
	startingSec := mins * 60
	if startingSec < totalSec {
		startingSec += 60 * 60
	}

	durFloat := math.Abs(startingSec - totalSec)

	//parse the duration into time.Duration
	return time.ParseDuration(fmt.Sprintf("%.0fs", durFloat))
}

func (hj hourlyJob) String() string {
	startingPoint := hj.startingPoint
	if startingPoint == "" {
		startingPoint = "immediately"
	}
	return fmt.Sprintf("[%s] %s", hj.getInterval(), startingPoint)
}

///////////////////////////////
////////// DAILY  ////////////
/////////////////////////////

//the interval is simply == 24 hours
//the starting point should be in hour and minute parseable in format hhmm
//e.g. "1530"
type dailyJob struct {
	startingPoint string
	timeLocation  *time.Location
}

func (dj dailyJob) getInterval() time.Duration { return time.Hour * 24 }

//calculate the sleep duration for daily category
func (dj dailyJob) getSleepDuration(startTime time.Time) (time.Duration, error) {
	var result time.Duration

	//return if starting point is empty string
	if dj.startingPoint == "" {
		return result, nil
	}

	//check whether starting point is into hour parseable
	if _, err := time.Parse("1504", dj.startingPoint); err != nil {
		return result, StartingPointError
	}

	timeThen, _ := time.ParseInLocation("20060102", startTime.Format("20060102"), startTime.Location())
	dur, _ := time.ParseDuration(fmt.Sprintf("%sh%sm", dj.startingPoint[:2], dj.startingPoint[2:]))
	//add the time difference between server and the selected time location
	timeThen = timeThen.Add(dur).In(dj.timeLocation).Add(calculateTimeDiff(dj.timeLocation))

	//add 1 day to timeThen if it's before time now
	if startTime.After(timeThen) {
		timeThen = timeThen.AddDate(0, 0, 1)
	}

	return timeThen.Sub(startTime), nil
}

func (dj dailyJob) String() string {
	startingPoint := dj.startingPoint
	if startingPoint == "" {
		startingPoint = "immediately"
	}
	return fmt.Sprintf("[%s] %s", dj.getInterval(), startingPoint)
}

///////////////////////////////
////////// WEEKLY ////////////
/////////////////////////////

//the interval is a week, so 7 * 24hours
//input weekday is in format of "weekday hour" separated by @ symbol (e.g. "Monday@1530")
type weeklyJob struct {
	startingPoint string
	timeLocation  *time.Location
}

//function to determine whether a weekday string is valid or not
//valid one is e.g. "monday" or "mOnDAY" (will be formatted using strings library)
//also return the number of seconds lapsed (e.g. for monday is 0 and for wednesday is 2 * 24 * 3600s)
//if not valid then return error
func getWeekDuration(weekday string) (time.Duration, error) {
	//format it to become the title case
	weekdayFormatted := strings.Title(strings.ToLower(weekday))

	var numDay int

	switch weekdayFormatted {
	case "Monday":
		numDay = 0
	case "Tuesday":
		numDay = 1
	case "Wednesday":
		numDay = 2
	case "Thursday":
		numDay = 3
	case "Friday":
		numDay = 4
	case "Saturday":
		numDay = 5
	case "Sunday":
		numDay = 6
	default:
		//return error if weekday is not a valid one
		return time.Second * 0, fmt.Errorf("Invalid weekday string: %s", weekday)
	}

	//parse duration using the number of days
	totalSec := numDay * 24 * 3600
	return time.ParseDuration(fmt.Sprintf("%ds", totalSec))
}

func (wj weeklyJob) getInterval() time.Duration { return time.Hour * 24 * 7 }

//calculate the sleep duration for weekly category
func (wj weeklyJob) getSleepDuration(startTime time.Time) (time.Duration, error) {
	var result time.Duration

	//return if starting point is empty string
	if wj.startingPoint == "" {
		return result, nil
	}

	//weekly string should contains exactly 2 elements after splitted by @
	weeklySplitted := strings.Split(wj.startingPoint, "@")
	if len(weeklySplitted) != 2 {
		return result, StartingPointError
	}

	//the idea is to substract the starting point converted in seconds by time now in seconds
	//monday 00:00 is the starting point
	//now create variable nowSec as total seconds from time now
	var nowSec float64
	wd := startTime.In(wj.timeLocation).Weekday().String()
	if wdDur, err := getWeekDuration(wd); err != nil {
		return result, err
	} else {
		nowSec += wdDur.Seconds()
	}

	//now format the time now into hours and minute to be converted into seconds
	//it is safe to assume that both of them are into float parseable
	//add the hours and minutes into nowSec
	hour, min := startTime.Format("15"), startTime.Format("04")
	curHour, _ := strconv.ParseFloat(hour, 64)
	curMin, _ := strconv.ParseFloat(min, 64)
	nowSec += curHour*3600 + curMin*60

	//now convert the starting point into seconds as well
	var thenSec float64
	if wdDur, err := getWeekDuration(weeklySplitted[0]); err != nil {
		return result, err
	} else {
		thenSec += wdDur.Seconds()
	}

	//make sure that the second part is into time parseable in format hhmm
	if _, err := time.Parse("1504", weeklySplitted[1]); err != nil {
		return result, err
	} else {
		//now it is safe to assume that it is into duration in this manner parseable
		dur, _ := time.ParseDuration(fmt.Sprintf("%sh%sm", weeklySplitted[1][:2], weeklySplitted[1][2:]))
		thenSec += dur.Seconds()
	}

	//also add the time difference
	thenSec += calculateTimeDiff(wj.timeLocation).Seconds()

	//add 1 week if thenSec is lesser than nowSec
	if thenSec < nowSec {
		thenSec += 7 * 24 * 3600
	}

	return time.ParseDuration(fmt.Sprintf("%.0fs", math.Abs(thenSec-nowSec)))
}

func (wj weeklyJob) String() string {
	startingPoint := wj.startingPoint
	if startingPoint == "" {
		startingPoint = "immediately"
	}
	return fmt.Sprintf("[%s] %s", wj.getInterval(), startingPoint)
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
