package treego

type Api struct {
	// Map of callbacks for specific events coming from TreeScale Host
	Callbacks map[string][]EventCallback
	// Token for this API
	// If not set, it would be generated during connect process
	Token string
	// API version for communication
	ApiVersion uint32

	// networking for handling API transport
	network Network
}

func (api *Api) handle_data(data []byte, endpoint string) {

}
