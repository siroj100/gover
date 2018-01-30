package gover

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

/*func TestCalculateTimeDiff(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	dur := calculateTimeDiff(loc)

	assert.Equal(t, time.Hour*6, dur)
}*/

func TestHourlyInterval(t *testing.T) {
	hj := hourlyJob{"123", globalTimeLoc}
	interval := hj.getInterval()
	assert.Equal(t, float64(3600), interval.Seconds())

	time1, _ := time.Parse("20060102 1504", "20160712 0050")
	_, err := hj.getSleepDuration(time1)
	assert.Equal(t, StartingPointError, err)

	hj.startingPoint = "60"
	_, err = hj.getSleepDuration(time1)
	assert.Equal(t, StartingPointError, err)

	hj = hourlyJob{"30", globalTimeLoc}
	timeDur, err := hj.getSleepDuration(time1)
	assert.NoError(t, err)
	assert.Equal(t, float64(40*60), timeDur.Seconds())

	time2, _ := time.Parse("20060102 1504", "20160712 0010")
	timeDur, err = hj.getSleepDuration(time2)
	assert.NoError(t, err)
	assert.Equal(t, float64(20*60), timeDur.Seconds())

}

func TestDailyInterval(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	dj := dailyJob{"1330", loc}
	interval := dj.getInterval()
	assert.Equal(t, float64(24*3600), interval.Seconds())

	timeDiff := calculateTimeDiff(loc)
	time1, _ := time.Parse("20060102 1504", "20160521 1000")
	dur, err := dj.getSleepDuration(time1.Add(timeDiff))
	assert.NoError(t, err)
	assert.Equal(t, float64(3.5*3600), dur.Seconds())

	time2, _ := time.Parse("20060102 1504", "20160521 1800")
	dur, err = dj.getSleepDuration(time2.Add(timeDiff))
	assert.NoError(t, err)
	assert.Equal(t, float64(19.5*3600), dur.Seconds())
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

func TestWeeklyInterval(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	timeDiff := calculateTimeDiff(loc)

	wj := weeklyJob{"Wednesday@1530", loc}
	timeNow, _ := time.Parse("2006-01-02 15:04", "2016-11-04 10:00") //Friday
	timeNow = timeNow.Add(timeDiff)

	dur, err := wj.getSleepDuration(timeNow)
	assert.NoError(t, err)
	assert.Equal(t, float64(5*24*3600+55*360), dur.Seconds())

	timeNow, _ = time.Parse("2006-01-02 15:04", "2016-11-01 10:00") //Tuesday
	timeNow = timeNow.Add(timeDiff)

	dur, err = wj.getSleepDuration(timeNow)
	assert.NoError(t, err)
	assert.Equal(t, float64(24*3600+55*360), dur.Seconds())
}
