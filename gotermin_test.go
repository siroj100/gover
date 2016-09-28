package gover

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
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
