package treego

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"sync"
	"time"
)

const (
	ERROR_EVENT_PARSE_DATA = 1
	DEFAULT_API_VERSION    = 1
)

type Api struct {
	// Map of callbacks for specific events coming from TreeScale Host
	Callbacks        map[string][]EventCallback
	callbacks_locker *sync.Mutex
	// Token for this API
	// If not set, it would be generated during connect process
	Token string
	// API version for communication
	ApiVersion uint32

	// networking for handling API transport
	network *Network

	// callback for handling errors
	onError func(int, error)

	// list of endpoints who are connected to this API
	endpoints []string
	// default endpoint for making communication with
	default_endpoint int
}

func NewApi(token string, endpoint ...string) *Api {
	api := &Api{
		Callbacks:        make(map[string][]EventCallback),
		callbacks_locker: &sync.Mutex{},
		Token:            token,
		ApiVersion:       DEFAULT_API_VERSION,
		network:          nil,
		onError:          func(c int, e error) {},
		endpoints:        endpoint,
		default_endpoint: 0,
	}

	api.network = newNetwork(api)
	return api
}

// Starting API client
func (api *Api) Start() {
	api.network.connect(api.endpoints[api.default_endpoint])
	for {
		time.Sleep(time.Millisecond * 1000)
	}
}

// Setting default endpoint for easy API usage
func (api *Api) SetDefaultEndpoint(endpoint string) {
	for i, ep := range api.endpoints {
		if ep == endpoint {
			api.default_endpoint = i
			return
		}
	}

	// if we got here then our endpoint doesn't exists in the list
	// so probably we need to add it and set as a default
	api.default_endpoint = len(api.endpoints)
	api.endpoints = append(api.endpoints, endpoint)
}

// Handling data from networking
func (api *Api) handle_data(data []byte, endpoint string) {
	event := &Event{}
	event.FromBytes(data)

	api.callbacks_locker.Lock()

	// triggering event callbacks based on parsed event
	if callbacks, ok := api.Callbacks[event.Name]; ok {
		for _, cb := range callbacks {
			// if callback returned false, just stopping the loop
			// it is something like e.preventDefault() from JS
			if !cb(event) {
				break
			}
		}
	}

	api.callbacks_locker.Unlock()
}

// Adding callback to specific event
func (api *Api) On(name string, callback EventCallback) {
	api.callbacks_locker.Lock()
	if _, ok := api.Callbacks[name]; !ok {
		api.Callbacks[name] = make([]EventCallback, 0)
	}

	api.Callbacks[name] = append(api.Callbacks[name], callback)
	api.callbacks_locker.Unlock()
}

// Removing callback function from specific event callbacks list
func (api *Api) Off(name string, callback EventCallback) {
	api.callbacks_locker.Lock()
	if callbacks, ok := api.Callbacks[name]; ok {
		for i, cb := range callbacks {
			// if we have this callback in our callback list
			// removing it from the list
			if reflect.ValueOf(cb).Pointer() == reflect.ValueOf(callback).Pointer() {
				callbacks = callbacks[:i+copy(callbacks[i:], callbacks[i+1:])]
				break
			}
		}
	}
	api.callbacks_locker.Unlock()
}

// Remove event based on name
func (api *Api) Rm(name string) {
	api.callbacks_locker.Lock()
	if _, ok := api.Callbacks[name]; ok {
		delete(api.Callbacks, name)
	}
	api.callbacks_locker.Unlock()
}

// Just a shortcut function for networking emit
func (api *Api) Emit(name string, data []byte, endpoint string) {
	// if endpoint is empty then we need to choose default endpoint to send
	if len(endpoint) == 0 {
		endpoint = api.endpoints[api.default_endpoint]
	}

	api.network.emit(endpoint, NewEvent(name, data))
}

// Emitting event with JSON encoding
func (api *Api) EmitJson(name string, data interface{}, endpoint string) error {
	buffer, err := json.Marshal(data)
	if err != nil {
		return err
	}

	api.Emit(name, buffer, endpoint)
	return nil
}

// Generating first handshake for sending it
func (api *Api) firstHandshake() []byte {
	token_data := []byte(api.Token)
	token_len := len(token_data)
	buffer_len := 4 + 4 + token_len + 8
	buffer := make([]byte, buffer_len)
	offset := 0

	// writing API Version
	binary.BigEndian.PutUint32(buffer[offset:offset+4], api.ApiVersion)
	offset += 4

	// Writing length of token and value
	binary.BigEndian.PutUint32(buffer[offset:offset+4], uint32(token_len+8))
	offset += 4
	// Writing token data
	copy(buffer[offset:offset+token_len], token_data)
	offset += token_len

	// writing value
	copy(buffer[offset:], DEFAULT_API_PATH)

	return buffer
}
