package stats

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Observations provides a means to record observations during program execution.
type Observations struct {
	lock sync.Mutex
	obs  map[string]interface{}
}

// Record records an observation.
func (o *Observations) Record(key string, val interface{}) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if o.obs == nil {
		o.obs = make(map[string]interface{})
	}

	if _, ok := o.obs[key]; ok {
		key = fmt.Sprintf("%s%s", key, time.Now())
	}
	o.obs[key] = val
}

// Marshal marshals all observations.
func (o *Observations) Marshal() (string, error) {
	if o.obs == nil {
		return "", nil
	}
	bytes, err := json.Marshal(o.obs)
	return string(bytes), err
}
