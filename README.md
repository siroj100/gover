##GOVER (only works for go version 1.7 and above)
###Current features:
- Cronjob like scheduler
- Customizeable auto retry function with timeout

##Usage

##1. Gotermin

The job function is a func(context.Context)  
The input context is only to handle condition where 1 interval has passed. It's perfectly OK to ignore it in function
```
type Cat struct{
	Name 	string
	Meow	string
}

func (c Cat) meowing(ctx context.Context){
	fmt.Printf("%s says '%s!!'\n", c.Name, c.Meow)
}
```

###1.1 hourly scheduler
This function will run on hourly basis. time location is required to make sure it's running in the correct timezone
```
moritz := Cat{"Moritz", "Rawrr"}
jkt, _ := time.LoadLocation("Asia/Jakarta")
hourly, _ := gover.NewHourly(moritz.meowing, "30", jkt)
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

###1.2 daily scheduler
In principal the same with hourly. Only the second parameter should the hour and minute in format hhmm (use empty string "" to run it immediately) 
```
daily, _ := gover.NewDaily(moritz.meowing, "0530", jkt)
if err := daily.Start(); err != nil{
	panic(err)
}
```

###1.3 custom interval
The interval this time is customizeable (it will run immediately every now and then)
```
interval := time.Second * 30
customInterval, _ := gover.NewCustomInterval(moritz.meowing, interval, jkt)
if err := customInterval.Start(); err != nil{
	panic(err)
}

```

##2. CrontabMinE
This is actually works as containers for gotermins  
Also has method Print() to return current conditions as string  

Example: 
```
//start with time location (all gotermins will follow this location)
berlin, _ := time.LoadLocation("Europe/Berlin")
crontab, err := gover.NewCrontab(berlin)
```

Register the schedulers (rules are quite the same as the previous)  
```
//register new hourly
addie := Cat{"Addison", "Meow"}
err := crontab.RegisterNewHourly("addie", addie.meowing, "30")

//register new daily
bill := Cat{"William", "Zzzz"}
err = crontab.RegisterNewDaily("bill", bill.meowing, "0300")

//register new custom interval
jane := Cat{"Jennifer", "Nyan"}
err = crontab.RegisterNewCustomInterval("jane", jane.meowing, time.Second * 10)
```

Each one can be started/stopped all at once or by key
```
//start by key
crontab.Start("addie")

//start all
crontab.StartAll()

//stop by key
crontab.Stop("jane")

//stop all
crontab.StopAll()

crontab.Start("bill")
```

Print the summary
```
fmt.Println(crontab.Print())
//will print something like this:
Summary
Key-----Interval-----StartingPoint-----Status
addie-----hourly-----30-----inactive
bill-----daily-----0300-----active
jane-----custom (10s)-----immediately-----inactive
```



##3. Gover

###create new gover struct. The inputs are: 
- time.Duration as the timeout duration for the job
- func(context.Context) error
Job is any function with input context.Context and error output.  
Be careful at handling the context key and value since they are both interface{}.

```
type Animal struct{
	Name
}

func (a *Animal) setName(c context.Context) error {
	if c.Value("name") != nil {
		a.Name = c.Value("name").(string)
		return nil
	}

	return fmt.Errorf("Invalid name context")
}

timeout := time.Second * 10

var cat Animal
gover, err := gover.New(timeout, cat.setName)
//set input 
gover.Context = context.WithValue(context.Background(), "name", "Mr. Meowingston")
```

###set optional parameters
```
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
```
###run the function
```
if err := gover.Run(); err == nil{
	fmt.Println(cat.Name)
}

//this should print "Mr. Meowingston"

```


