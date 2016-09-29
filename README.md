##GOVER (only works for go version 1.7 and above)
###Current features:
- Cronjob like scheduler
- Customizeable auto retry function with timeout

##Usage

##1. Gotermin

###the job function is a func(context.Context)
###The input context is only to handle condition where 1 interval has passed. It's perfectly OK to ignore it in function
```
job := func(ctx context.Context){
	fmt.Println("foo")
}
```

##1.1 hourly scheduler
###this function will run on hourly basis. time location is required to make sure it's running in the correct timezone
```
jkt, _ := time.LoadLocation("Asia/Jakarta")
hourly, _ := gover.NewHourly(job, "30", jkt)
//"30" means that it will run every minute 30 every hour
//if not parseable then it will return an error
//exception is empty string "", which means it will run immediately 

//run the scheduler
if err := hourly.Start(); err != nil{
	panic(err)
}
//stop the scheduler at need
hourly.Stop()
```

##1.2 daily scheduler
###in principal the same with hourly. Only the second parameter should the hour and minute in format hhmm (use empty string "" to run it immediately) 
```
daily, _ := gover.NewDaily(job, "0530", jkt)
if err := daily.Start(); err != nil{
	panic(err)
}
```

##1.3 custom interval
###the interval this time is customizeable (it will run immediately every now and then)
```
interval := time.Second * 30
customInterval, _ := gover.NewCustomInterval(job, interval, jkt)
if err := customInterval.Start(); err != nil{
	panic(err)
}

```


##2. Gover

###create new gover struct. The inputs are: 
- time.Duration as the timeout duration for the job
- func(context.Context) (context.Context, error)
###job is any function with input context.Context and output (context.Context, error). Be careful at handling the context key and value since they are both interface{}. The purpose is to generalize all fuction, e.g.

```
timeout := time.Second * 10
job := func(ctx context.Context) (context.Context, error){
	//get input from context
	input := ctx.Value("input").(int)
	
	//set output in context
	return context.WithValue(ctx, "output", input + 1), nil
}

gover, err := gover.New(timeout, job)
//set input 
gover.Context = context.WithValue(context.Background(), "input", 1)

//set optional parameters

//how many times this function will be done again if error is returned
gover.MaxRetry = 3 
//keyword for error message that's not supposed to be retried
gover.NoRetryConditions = []string{"foo"}
//interval between each retrial
//if not specified or not parseable into time.Duration it will be retried immediately
gover.RetryInterval = "100ms"
//timeout for each retry 
//if not specified or not parseable into time.Duration it will not have a timeout 
gover.JobInterval = "1s"

//run the function
//read the output from context
var output int
if err := gover.Run(); err != nil{
	output = gover.Context.Value("output").(int)
}
//this should print 2
fmt.Println(output)

```




