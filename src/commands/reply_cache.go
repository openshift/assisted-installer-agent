package commands

import (
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/openshift/assisted-service/models"
	"github.com/thoas/go-funk"
)

func newCache() *cache.Cache {
	return cache.New(time.Hour, time.Hour)
}

func alreadyExistsInService(c *cache.Cache, stepType models.StepType, value string) bool {
	storedValue, ok := c.Get(string(stepType))
	return ok && funk.Equal(storedValue, value)
}

func storeInCache(c *cache.Cache, stepType models.StepType, value string) {
	c.Set(string(stepType), value, cache.DefaultExpiration)
}

func invalidateCache(c *cache.Cache) {
	c.Flush()
}
