package treego

type Event struct {
	// this path is used only internally
	// and would be 0 for all cases
	path []byte
	// Event name
	Name string
	// ID or name where this event is coming from
	From string
	// target ID or name, or maybe some hidden parameter
	Target string
	// Data for delivering with this event
	Data []byte
}

// Event callback function type
type EventCallback func(*Event) bool

func (e *Event) ToBytes() []byte {

}
