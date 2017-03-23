package treego

import (
	"encoding/binary"
	"github.com/satori/go.uuid"
	"reflect"
	"sync"
	"time"
)

const (
	// List of Endpoint types available to connect to
	ENDPOINT_TCP        EndpointType = 1
	DEFAULT_API_VERSION uint32       = 1

	MAX_API_VERSION uint32 = 1000

	// Maximum network message length is 30MB
	MAX_READ_DATA_LEN uint32 = 30000000
)

type EndpointType int

var (
	TMP_DATA = []byte{}
)

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

	ApiVersion uint32

	endpoint_token       string
	endpoint_value       uint64
	endpoint_info_locker *sync.Mutex
}

// Making new TreeApi object, if token is empty string
// Then we will generate it using uuid4
func NewApi(endpoint, token string, endpoint_type EndpointType) *TreeApi {
	if len(token) == 0 {
		token = uuid.NewV4().String()
	}

	ta := &TreeApi{
		endpoint:             endpoint,
		endpoint_type:        endpoint_type,
		tcp_network:          nil,
		callbacks:            make(map[string][]EventCallback),
		Token:                token,
		ApiVersion:           DEFAULT_API_VERSION,
		callbacks_locker:     &sync.Mutex{},
		endpoint_token:       "",
		endpoint_value:       uint64(0),
		endpoint_info_locker: &sync.Mutex{},
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

// Starting Loop and connecting to given endpoint
// channels - home many concurrent connections we will keep with endpoint
// More connections will get more Memory but will increase network performance
// if channels is <= 0 , then it will be set to 1
func (api *TreeApi) Start(channels int) {
	if channels <= 0 {
		channels = 1
	}

	api.tcp_network.connect(channels)

	for {
		time.Sleep(time.Second * 3)
	}
}

// Generating first handshake for sending it
func (api *TreeApi) firstHandshake() []byte {
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

// handling data from networking, this would be parsed as an event
func (api *TreeApi) handle_data(data []byte) {
	ev := new(Event)
	ev.FromBytes(data)
	api.Trigger(ev)
}

// Sending event to endpoint
func (api *TreeApi) Emit(event *Event) error {
	return api.tcp_network.write(event.ToBytes(), nil)
}
