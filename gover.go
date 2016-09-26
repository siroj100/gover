package gover

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Gover struct {
	//the function that's supposed to be run
	//input and output contain context here
	//instead of running interface{} context can also serve as input and output variables
	//only have to be cautious not to get mistaken by ambiguous key
	Job func(context.Context) (context.Context, error)
	//context with its cancel function
	Context context.Context
	Cancel  context.CancelFunc
	//deadline when we are supposed to be stop trying
	//this is mandatory to prevent the job running uncontrolably
	Deadline time.Time
	//number of maximum retry
	//if job returns an error then it will keep retrying until this number is exceeded
	MaxRetry int
	//this contains array of strings
	//strings contains will be used, so it doesn't have to be precised
	//however be careful to ambiguity
	//if empty then every error will be retried
	NoRetryConditions []string
	//specify the retry interval
	//this is a string that supposed to be parsed into time.Duration
	//if it fails to parse then the timeout for retry will be set into a very short duration
	RetryInterval string
	//specify the timeout for each jobs
	JobInterval string
}

func New(timeout time.Duration, job func(context.Context) (context.Context, error)) (*Gover, error) {
	//timeout can't be lower than 1ns
	if timeout.Nanoseconds() == int64(0) {
		return nil, fmt.Errorf("Timeout %.0fs is too short", timeout.Seconds())
	}

	return &Gover{
		Context:  context.Background(),
		Job:      job,
		Deadline: time.Now().Add(timeout),
	}, nil

}

func (g *Gover) Run() error {
	//return immediately if deadline is already exceeded
	if g.Deadline.Before(time.Now()) {
		return fmt.Errorf("Deadline %+v is already exceeded", g.Deadline)
	}

	//check the context
	//if it's not defined then simply use context.Background()
	if g.Context == nil {
		g.Context = context.Background()
	}

	//set deadline
	g.Context, g.Cancel = context.WithDeadline(g.Context, g.Deadline)

	return g.runWithTimeout()
}

func (g *Gover) runWithTimeout() error {
	var currentRetry int
	var err error

	retryChan := make(chan int, 1)
	errorChan := make(chan error, 1)

	doTheJob := func(retryNum int, child context.Context) {
		//update the parent context everytime the job is done
		g.Context, err = g.Job(g.Context)

		//check whether the child context is already done or not
		//if it is error then do nothing, just return
		if child.Err() != nil {
			return
		}

		if err != nil {
			//if error then this might should be retried
			//first check whether the error code is in no retry list
			for _, con := range g.NoRetryConditions {
				if strings.Contains(err.Error(), con) {
					errorChan <- fmt.Errorf("Error contains keyword: %s", con)
					return
				}
			}

			//then check if the retry number already exceeded
			//if that's the case then just return
			if retryNum >= g.MaxRetry {
				errorChan <- fmt.Errorf("Maximum number of retry exceeded")
				return
			}

			//otherwise fill the retry channel
			retryChan <- retryNum
			return
		}
		//if there's no error simply fill the error channel with nil
		errorChan <- nil
	}

	//do the job until it's done or expired
	for {
		//set retry interval here
		//default value is 1ms only, change only if parsing duration returns no error
		retryInterval := time.Millisecond
		if rt, err := time.ParseDuration(g.RetryInterval); err == nil {
			retryInterval = rt
		}

		//create child context
		//if jobinterval is stated then use different interval
		//otherwise derivate it from the parent
		var childCtx context.Context
		if jobInterval, err := time.ParseDuration(g.JobInterval); err != nil {
			childCtx, _ = context.WithCancel(g.Context)
		} else {
			childCtx, _ = context.WithTimeout(g.Context, jobInterval)
		}

		go doTheJob(currentRetry, childCtx)
		select {
		case <-g.Context.Done():
			//in this case the context is already cancelled
			//return error immediately and abandon the currently running go routine
			return fmt.Errorf("Context timeout")
		case err := <-errorChan:
			//return whatever error there is in this channel
			return err
		case retryNum := <-retryChan:
			//in this case continue the loop
			//update the currentRetry with 1 + retryNum
			currentRetry = retryNum + 1

			//sleep for the set interval before retrying
			time.Sleep(retryInterval)
			continue
		case <-childCtx.Done():
			//in this case child is timed out
			//return error if max retry is exceeded
			if currentRetry >= g.MaxRetry {
				return fmt.Errorf("Maximum number of retry exceeded")
			}
			//otherwise retry
			currentRetry += 1
			time.Sleep(retryInterval)
			continue
		}
	}

	return fmt.Errorf("Unknown error")
}
