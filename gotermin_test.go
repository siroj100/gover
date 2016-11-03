package gover

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
	"time"
)

func TestCreateNewHourly(t *testing.T) {
	job := func(ctx context.Context) {
		fmt.Println("foo")
	}

	invalidMinute := "foo"
	jkt, _ := time.LoadLocation("Asia/Jakarta")

	_, err := NewHourly(job, invalidMinute, jkt)
	assert.Error(t, err)

	_, err = NewHourly(job, "", jkt)
	assert.NoError(t, err)

	validMinute := "05"
	result, err := NewHourly(job, validMinute, jkt)
	assert.NoError(t, err)
	assert.Equal(t, validMinute, result.startingPoint)
	assert.Equal(t, "hourly", result.intervalCategory)

	_, err = NewHourly(job, validMinute, nil)
	assert.Error(t, err)
}

func TestCreateNewDaily(t *testing.T) {
	job := func(ctx context.Context) {
		fmt.Println("foo")
	}

	invalidHour := "foo"
	jkt, _ := time.LoadLocation("Asia/Jakarta")

	_, err := NewDaily(job, invalidHour, jkt)
	assert.Error(t, err)

	_, err = NewDaily(job, "", jkt)
	assert.NoError(t, err)

	validHour := "0530"
	result, err := NewDaily(job, validHour, jkt)
	assert.NoError(t, err)
	assert.Equal(t, validHour, result.startingPoint)
	assert.Equal(t, "daily", result.intervalCategory)

	_, err = NewDaily(job, validHour, nil)
	assert.Error(t, err)
}

func TestGetWeekDuration(t *testing.T) {
	dur, err := getWeekDuration("monDaY")
	assert.NoError(t, err)
	assert.Equal(t, float64(0), dur.Seconds())

	_, err = getWeekDuration("seniN")
	assert.Error(t, err)

	dur, err = getWeekDuration("friday")
	assert.NoError(t, err)
	assert.Equal(t, float64(4*24*3600), dur.Seconds())
}

func TestCreateNewWeekly(t *testing.T) {
	randomFunc := func(ctx context.Context) { fmt.Println("foo") }
	loc, _ := time.LoadLocation("Asia/Jakarta")
	_, err := NewWeekly(randomFunc, "Monday 1530", loc)
	assert.Error(t, err)

	_, err = NewWeekly(randomFunc, "Monday@1530", nil)
	assert.Error(t, err)

	_, err = NewWeekly(randomFunc, "Mondayz@1504", loc)
	assert.Error(t, err)

	_, err = NewWeekly(randomFunc, "Monday@1561", loc)
	assert.Error(t, err)

	gt, err := NewWeekly(randomFunc, "Monday@1530", loc)
	assert.NoError(t, err)
	assert.Equal(t, gt.startingPoint, "Monday@1530")
}

func TestCreateDurationUntilFirstJob(t *testing.T) {
	job := func(ctx context.Context) {
		fmt.Println("foo")
	}
	jkt, _ := time.LoadLocation("Asia/Jakarta")
	result, _ := NewHourly(job, "30", jkt)

	dur, err := result.durationUntilFirst()
	assert.NoError(t, err)
	seconds := dur.Seconds()
	assert.NotEqual(t, float64(0), seconds)
	result.startingPoint = "0530"
	_, err = result.durationUntilFirst()
	assert.Error(t, err)

	result, _ = NewDaily(job, "0530", jkt)

	dur, err = result.durationUntilFirst()
	assert.NoError(t, err)
	seconds = dur.Seconds()
	assert.NotEqual(t, float64(0), seconds)

	result.startingPoint = "30"
	_, err = result.durationUntilFirst()
	assert.Error(t, err)
}

func TestStartAndStopJob(t *testing.T) {
	c := make(chan interface{}, 1)

	job := func(ctx context.Context) {
		<-ctx.Done()
		c <- "foo"
	}

	jkt, _ := time.LoadLocation("Asia/Jakarta")
	result, _ := NewHourly(job, "", jkt)
	//stopping the job immediately should result in error
	err := result.Stop()
	assert.Error(t, err)

	//starting should work just fine
	err = result.Start()
	assert.NoError(t, err)

	//wait for 500ms
	time.Sleep(time.Millisecond * 500)

	//then try to stop it
	err = result.Stop()
	assert.NoError(t, err)

	//wait for 500ms
	time.Sleep(time.Millisecond * 500)

	assert.Equal(t, false, result.isActive)
	assert.Equal(t, "foo", <-c)
}

func TestCustomInterval(t *testing.T) {
	initial := int64(0)
	fmt.Println("initial number", initial)
	job := func(ctx context.Context) {
		atomic.AddInt64(&initial, 1)
		fmt.Println("current number: ", initial)
	}

	jkt, _ := time.LoadLocation("Asia/Jakarta")
	customInterval := time.Millisecond

	_, err := NewCustomInterval(job, customInterval, jkt)
	assert.Error(t, err)

	customInterval = time.Second

	result, err := NewCustomInterval(job, customInterval, jkt)
	assert.NoError(t, err)
	assert.Equal(t, "custom", result.intervalCategory)
	assert.Equal(t, "", result.startingPoint)
	assert.Equal(t, jkt, result.timeLocation)
	assert.Equal(t, customInterval, result.customInterval)

	err = result.Start()
	assert.NoError(t, err)
	//sleep for 3 seconds
	//then the initial should be 3 afterwards
	sleepTime := int64(3)

	sleepDur, _ := time.ParseDuration(fmt.Sprintf("%ds", sleepTime))
	time.Sleep(sleepDur)

	err = result.Stop()
	assert.NoError(t, err)
	assert.Equal(t, sleepTime, initial)
}

func TestCalculateTimeDiff(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	dur := calculateTimeDiff(loc)

	assert.Equal(t, time.Hour*6, dur)
}

func randomFunc(ctx context.Context) { fmt.Println("foo") }

func TestCalculateHourlyDuration(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")

	gt, _ := NewHourly(randomFunc, "20", loc)
	timeNow, _ := time.Parse("2006-01-02 15:04", "2016-05-21 00:30")
	dur, err := gt.calculateHourlyDuration(timeNow)
	assert.NoError(t, err)
	assert.Equal(t, float64(50*60), dur.Seconds())

	timeNow, _ = time.Parse("2006-01-02 15:04", "2016-05-21 00:05")
	dur, err = gt.calculateHourlyDuration(timeNow)
	assert.NoError(t, err)
	assert.Equal(t, float64(15*60), dur.Seconds())
}

func TestCalculateDailyDuration(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	timeDiff := calculateTimeDiff(loc)

	gt, _ := NewDaily(randomFunc, "1330", loc)
	timeNow, _ := time.Parse("2006-01-02 15:04", "2016-05-21 10:00")
	timeNow = timeNow.Add(timeDiff)

	dur, err := gt.calculateDailyDuration(timeNow)
	assert.NoError(t, err)
	assert.Equal(t, float64(3.5*3600), dur.Seconds())

	timeNow, _ = time.Parse("2006-01-02 15:04", "2016-05-21 18:00")
	timeNow = timeNow.Add(timeDiff)
	dur, err = gt.calculateDailyDuration(timeNow)
	assert.NoError(t, err)
	assert.Equal(t, float64(19.5*3600), dur.Seconds())
}

func TestCalculateWeeklyDuration(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	timeDiff := calculateTimeDiff(loc)

	gt, _ := NewWeekly(randomFunc, "Wednesday@1530", loc)
	timeNow, _ := time.Parse("2006-01-02 15:04", "2016-11-04 10:00") //Friday
	timeNow = timeNow.Add(timeDiff)

	dur, err := gt.calculateWeeklyDuration(timeNow)
	assert.NoError(t, err)
	assert.Equal(t, float64(5*24*3600+55*360), dur.Seconds())

	timeNow, _ = time.Parse("2006-01-02 15:04", "2016-11-01 10:00") //Tuesday
	timeNow = timeNow.Add(timeDiff)

	dur, err = gt.calculateWeeklyDuration(timeNow)
	assert.NoError(t, err)
	assert.Equal(t, float64(24*3600+55*360), dur.Seconds())
}
