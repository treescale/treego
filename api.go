package treego

import (
	"github.com/satori/go.uuid"
	"reflect"
	"sync"
)

const (
	// List of Endpoint types available to connect to
	ENDPOINT_TCP EndpointType = 1
)

type EndpointType int

type TreeApi struct {
	// endpoint to connect to
	endpoint      string
	endpoint_type EndpointType

	// tcp_networking reference
	tcp_network *tcp_net

	// Callbacks to trigger for specific events
	callbacks        map[string][]EventCallback
	callbacks_locker *sync.Mutex

	// Token/Name for current API
	// This would be auto-generated if not set
	// from uuid4
	Token string
}

// Making new TreeApi object, if token is empty string
// Then we will generate it using uuid4
func NewApi(endpoint, token string, endpoint_type EndpointType) *TreeApi {
	if len(token) == 0 {
		token = uuid.NewV4().String()
	}

	ta := &TreeApi{
		endpoint:      endpoint,
		endpoint_type: endpoint_type,
		tcp_network:   nil,
		callbacks:     make(map[string][]EventCallback),
		Token:         token,
	}

	ta.tcp_network = newTcpNet(ta)
	return ta
}

// Setting event callbacks
func (api *TreeApi) On(name string, callback EventCallback) {
	api.callbacks_locker.Lock()
	if _, ok := api.callbacks[name]; !ok {
		api.callbacks[name] = make([]EventCallback, 0)
	}

	api.callbacks[name] = append(api.callbacks[name], callback)
	api.callbacks_locker.Unlock()
}

// Removing single callback from event
func (api *TreeApi) Off(name string, callback EventCallback) {
	api.callbacks_locker.Lock()
	// if we don't have event with this name, just returning
	if cbs, ok := api.callbacks[name]; ok {
		callback_pointer := reflect.ValueOf(callback).Pointer()

		for i, cb := range cbs {
			if reflect.ValueOf(cb).Pointer() == callback_pointer {
				cbs = append(cbs[:i], cbs[i+1:]...)
				break
			}
		}
	}

	api.callbacks_locker.Unlock()
}

// Removing event from callback list
func (api *TreeApi) RmEvent(name string) {
	api.callbacks_locker.Lock()
	if _, ok := api.callbacks[name]; ok {
		delete(api.callbacks, name)
	}
	api.callbacks_locker.Unlock()
}

// triggering event based on event object
func (api *TreeApi) Trigger(event *Event) {
	api.callbacks_locker.Lock()

	if cbs, ok := api.callbacks[event.Name]; ok {
		for _, cb := range cbs {
			if !cb(event) {
				break
			}
		}
	}

	api.callbacks_locker.Unlock()
}

// triggering local event just based on Data
func (api *TreeApi) trigger_local(name string, data []byte) {
	api.Trigger(newEvent(name, data))
}
