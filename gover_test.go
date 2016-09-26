package gover

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGover(t *testing.T) {
	var foo, foo2 string

	ctx := context.WithValue(context.Background(), "foo", "bar")
	ctx = context.WithValue(ctx, "foo2", "bar2")

	testFunc := func(c context.Context) (context.Context, error) {
		foo = c.Value("foo").(string)
		foo2 = c.Value("foo2").(string)
		return context.WithValue(c, "foo3", "bar3"), nil
	}

	var timeNull time.Duration
	_, err := New(timeNull, testFunc)
	assert.Error(t, err)

	gover, err := New(time.Second*5, testFunc)
	assert.NoError(t, err)
	gover.Context = ctx
	err = gover.Run()
	assert.NoError(t, err)
	assert.Equal(t, "bar", foo)
	assert.Equal(t, "bar2", foo2)
	assert.Equal(t, "bar3", gover.Context.Value("foo3"))
}

func TestRetryFunctionality(t *testing.T) {
	//test the max retry functionality
	initNum := 0
	testFunc := func(c context.Context) (context.Context, error) {
		if initNum < 3 {
			initNum += 1
			return c, fmt.Errorf("should be at least 3")
		}
		return c, nil
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
	timeoutFunc := func(ctx context.Context) (context.Context, error) {
		tryNum += 1
		to, _ := time.ParseDuration(fmt.Sprintf("%dms", initialTO))
		initialTO = initialTO - 100
		time.Sleep(to)
		return ctx, err
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
