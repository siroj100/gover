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
	gover, err = New(time.Hour, testFunc)
	gover.MaxRetry = 3
	err = gover.Run()
	assert.NoError(t, err)

	//test the retry interval
	initNum = 0
	gover, err = New(time.Hour, testFunc)
	gover.MaxRetry = 3
	gover.RetryInterval = time.Millisecond * 100
	timeNow := time.Now()
	err = gover.Run()
	elapsedTime := time.Since(timeNow).Seconds()
	assert.NoError(t, err)

	//elapsed time should be about 300ms
	isMoreThan300 := elapsedTime > 0.3
	isLessThan400 := elapsedTime < 0.4

	assert.Equal(t, true, isMoreThan300)
	assert.Equal(t, true, isLessThan400)

	//test the no retry conditions
	initNum = 0
	gover, err = New(time.Hour, testFunc)
	gover.MaxRetry = 3
	gover.NoRetryConditions = []string{"should"}
	err = gover.Run()
	assert.Error(t, err)

	/*//gradually reduce the func duration
	initial := 5200

	testFunc := func(c context.Context) (context.Context, error){
		dur, _ := time.ParseDuration(fmt.Sprintf("%dms", initial))
		initial = initial - 500
		time.Sleep(dur)
		return c, nil
	}

	//set timeout duration into 3s
	//max retry to 5
	gover, err := New(time.Hour, testFunc)
	assert.NoError(t, err)
	gover.MaxRetry = 5
	gover.JobTimeout = time.Second * 3
	err = gover.Run()
	assert.NoError(t, err)
	//initial supposed to be successful when it hit 2700
	assert.Equal(t, 2700, initial)
	*/

}
