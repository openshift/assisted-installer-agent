package commands

import (
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/openshift/assisted-service/models"
	"github.com/thoas/go-funk"
)

var replyCache = cache.New(time.Hour, time.Hour)

func alreadyExistsInService(stepType models.StepType, value string) bool {
	storedValue, ok := replyCache.Get(string(stepType))
	return ok && funk.Equal(storedValue, value)
}

func storeInCache(stepType models.StepType, value string) {
	replyCache.Set(string(stepType), value, cache.DefaultExpiration)
}
