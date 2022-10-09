// this package can be copied when creating new microservice
package internal

import (
	"fmt"
	"sync"

	"github.com/iamthe1whoknocks/saiService"
)

// Internal configuration should be global (specified here handler elements will have access to it)
type Configuration struct {
	Mutex *sync.RWMutex
	Cfg   map[string]interface{}
}

var (
	MicroserviceConfiguration *Configuration = nil // initialize global configuration
)

func InitLocalConfiguration(svc saiService.Service) {
	//initialize and set localConfiguration
	MicroserviceConfiguration.Set(svc.Configuration)
}

// Initialize + get method for global local configuration
func NewConfiguration() *Configuration {
	MicroserviceConfiguration = &Configuration{
		Mutex: new(sync.RWMutex),
		Cfg:   make(map[string]interface{}),
	}

	return MicroserviceConfiguration
}

// Set microservice local configuration
func (c *Configuration) Set(configuration map[string]interface{}) {
	c.Mutex.Lock()
	c.Cfg = configuration
	c.Mutex.Unlock()
}

// Get value from microservice local configuration
func (c *Configuration) getValue(key string) interface{} {
	c.Mutex.RLock()
	val, ok := c.Cfg[key]
	if !ok {
		return fmt.Errorf("key %s was not found", key)
	}
	c.Mutex.RUnlock()
	return val
}
