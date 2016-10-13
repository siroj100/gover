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

	assert.Equal(t, time.Hour*-5, dur)
}
