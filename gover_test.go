package gover

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type Animal struct {
	Name string
}

func (a *Animal) setName(c context.Context) error {
	if c.Value("name") != nil {
		a.Name = c.Value("name").(string)
		return nil
	}

	return fmt.Errorf("Invalid name context")
}

func TestGover(t *testing.T) {
	ctx := context.WithValue(context.Background(), "name", "Mr. Meowingston")
	var cat Animal

	var timeNull time.Duration
	_, err := New(timeNull, cat.setName)
	assert.Error(t, err)

	gover, err := New(time.Second*5, cat.setName)
	assert.NoError(t, err)
	gover.Context = ctx
	err = gover.Run()
	assert.NoError(t, err)
	assert.Equal(t, "Mr. Meowingston", cat.Name)
}

func TestRetryFunctionality(t *testing.T) {
	//test the max retry functionality
	initNum := 0
	testFunc := func(c context.Context) error {
		if initNum < 3 {
			initNum += 1
			return fmt.Errorf("should be at least 3")
		}
		return nil
	}

	gover, err := New(time.Hour, testFunc)
	assert.NoError(t, err)
	gover.MaxRetry = 2
	err = gover.Run()
	assert.Error(t, err)

	initNum = 0
	gover.MaxRetry = 3
	err = gover.Run()
	assert.NoError(t, err)

	//test the retry interval
	initNum = 0
	gover.MaxRetry = 3
	gover.RetryInterval = "100ms"
	timeNow := time.Now()
	err = gover.Run()
	elapsedTime := time.Since(timeNow).Seconds()
	assert.NoError(t, err)

	//elapsed time should be about 300ms
	isMoreThan300 := elapsedTime > 0.3
	isLessThan400 := elapsedTime < 0.4

	assert.Equal(t, true, isMoreThan300)
	assert.Equal(t, true, isLessThan400)

	//set invalid retry interval
	initNum = 0
	gover.RetryInterval = "100zs"
	timeNow = time.Now()
	err = gover.Run()
	elapsedTime = time.Since(timeNow).Seconds()
	assert.NoError(t, err)

	//elapsed time should be less than 10ms
	isLessThan10 := elapsedTime < 0.01
	assert.Equal(t, true, isLessThan10)

	//test the no retry conditions
	initNum = 0
	gover.MaxRetry = 3
	gover.NoRetryConditions = []string{"should"}
	err = gover.Run()
	assert.Error(t, err)

	//test job interval
	//create new function with decrementing timeout
	//initial timeout is 520ms, decrement 100ms each time it's called
	initialTO := 520
	tryNum := 0
	timeoutFunc := func(ctx context.Context) error {
		tryNum += 1
		to, _ := time.ParseDuration(fmt.Sprintf("%dms", initialTO))
		initialTO = initialTO - 100
		time.Sleep(to)
		return err
	}

	gover, err = New(time.Hour, timeoutFunc)
	assert.NoError(t, err)
	gover.MaxRetry = 5
	//set job interval into only 300ms
	//so the first (520ms), second(420ms) and third(320ms) should fail
	//we expect it to succed on the 4th trial
	gover.JobInterval = "300ms"
	err = gover.Run()
	assert.NoError(t, err)
	assert.Equal(t, 4, tryNum)

	//now test the max retry and job interval
	initialTO = 520
	tryNum = 0
	gover.JobInterval = "1ns" //this is practically impossible to pass
	err = gover.Run()
	assert.Error(t, err)
	assert.Equal(t, tryNum, gover.MaxRetry)

	//now test collision between job interval and the parent context timeout
	gover, _ = New(time.Second*1000, timeoutFunc)
	initialTO = 520
	tryNum = 0
	gover.JobInterval = "300ms"
	//previously it returns 4, now it should be less and returns an error
	err = gover.Run()
	assert.Error(t, err)
	assert.Equal(t, 2, tryNum)
}
