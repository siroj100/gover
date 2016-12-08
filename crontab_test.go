package gover

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
	"time"
)

func TestCreateNewCrontab(t *testing.T) {
	_, err := NewCrontab(nil)
	assert.Error(t, err)

	jkt, _ := time.LoadLocation("Asia/Jakarta")
	result, err := NewCrontab(jkt)
	assert.NoError(t, err)
	assert.Equal(t, jkt, result.timeLocation)
}

type Cat struct {
	Name string
	Race string
	Age  int64
}

func (c *Cat) meowing(ctx context.Context) {
	fmt.Printf("%s a %d old %s cat says 'Meow!'\n'", c.Name, c.Age, c.Race)
}

func (c *Cat) aging(ctx context.Context) {
	currentAge := c.Age
	newAge := atomic.AddInt64(&currentAge, 1)
	c.Age = newAge
}

func TestRegisterAndStartNew(t *testing.T) {
	jkt, _ := time.LoadLocation("Asia/Jakarta")

	crontab, _ := NewCrontab(jkt)
	defer crontab.StopAll()

	addie := Cat{"Addie", "Siammese", 3}

	err := crontab.RegisterNewHourly("addie", addie.meowing, "123")
	assert.Error(t, err)

	err = crontab.RegisterNewHourly("addie", addie.meowing, "01")
	assert.NoError(t, err)

	err = crontab.RegisterNewHourly("addie", addie.meowing, "01")
	assert.Error(t, err)

	lorrie := Cat{"Lorrie", "Russian Blue", 5}
	err = crontab.RegisterNewHourly("lorrie", lorrie.meowing, "")
	assert.NoError(t, err)

	err = crontab.Start("eddie")
	assert.Error(t, err)

	err = crontab.StartAll()
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	err = crontab.Start("addie")
	assert.Error(t, err)

	err = crontab.Stop("lorry")
	assert.Error(t, err)

	err = crontab.Stop("lorrie")
	assert.NoError(t, err)

	crontab.StopAll()

	lowell := Cat{"Lowell", "Amber", 7}
	err = crontab.RegisterNewDaily("lowell", lowell.meowing, "01")
	assert.Error(t, err)

	err = crontab.RegisterNewDaily("lowell", lowell.meowing, "0530")
	assert.NoError(t, err)

	err = crontab.RegisterNewDaily("lorrie", lowell.meowing, "0530")
	assert.Error(t, err)

	err = crontab.RegisterNewDaily("lowell2", lowell.meowing, "")
	assert.NoError(t, err)

	err = crontab.Start("lowell2")
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	err = crontab.Start("lowell2")
	assert.Error(t, err)

	moritz := Cat{"Moritz", "Bengal", 1}
	err = crontab.RegisterNewCustomInterval("lowell", moritz.meowing, time.Second*2)
	assert.Error(t, err)

	err = crontab.RegisterNewCustomInterval("moritz", moritz.meowing, time.Second*2)
	assert.NoError(t, err)

	err = crontab.RegisterNewCustomInterval("moritz2", moritz.meowing, time.Millisecond*999)
	assert.Error(t, err)

	err = crontab.Start("moritz")
	assert.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	err = crontab.Start("moritz")
	assert.Error(t, err)

	crontab.StopAll()

	time.Sleep(time.Millisecond * 500)

	err = crontab.RegisterNewCustomInterval("addie_age", addie.aging, time.Second)
	assert.NoError(t, err)

	err = crontab.RegisterNewCustomInterval("moritz_age", moritz.aging, time.Second*2)
	assert.NoError(t, err)

	err = crontab.Start("addie_age")
	assert.NoError(t, err)

	err = crontab.Start("moritz_age")
	assert.NoError(t, err)

	time.Sleep(time.Second * 3)

	err = crontab.Stop("addie_age")
	assert.NoError(t, err)

	time.Sleep(time.Second * 2)

	err = crontab.Stop("moritz_age")
	assert.NoError(t, err)

	assert.Equal(t, int64(6), addie.Age)
	assert.Equal(t, int64(4), moritz.Age)

	assert.Equal(t, 7, len(crontab.cronjobs))

	<-time.After(time.Millisecond * 100)

	fmt.Println(crontab)

	crontab.Start("moritz_age")

	<-time.After(time.Millisecond * 100)

	//test get keys
	allkeys := crontab.GetAllKeys()
	assert.Equal(t, 7, len(allkeys))

	activeKeys := crontab.GetActiveKeys()
	assert.Equal(t, 1, len(activeKeys))

	inactiveKeys := crontab.GetInactiveKeys()
	assert.Equal(t, 6, len(inactiveKeys))

	_, err = crontab.GetCronjob("foo")
	assert.Equal(t, KeyNotFoundError, err)

	gt, err := crontab.GetCronjob("moritz_age")
	assert.NoError(t, err)
	assert.Equal(t, true, gt.isActive)
}
