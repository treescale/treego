package treego

import "encoding/binary"

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

var DEFAULT_API_PATH = []byte{0, 0, 0, 0, 0, 0, 0, 0}

func NewEvent(name string, data []byte) *Event {
	return &Event{
		path:   DEFAULT_API_PATH,
		Name:   name,
		From:   "",
		Target: "",
		Data:   data,
	}
}

func (e *Event) ToBytes() []byte {
	name_data := []byte(e.Name)
	from_data := []byte(e.From)
	target_data := []byte(e.Target)
	data_len := 4 + 8 + // 4 bytes for path length
		4 + len(name_data) +
		4 + len(from_data) +
		4 + len(target_data) +
		len(e.Data)

	buffer := make([]byte, data_len+4) // adding more 4 bytes for writing total data length
	offset := 0
	// adding total data length
	binary.BigEndian.PutUint32(buffer[offset:offset+4], uint32(data_len))
	offset += 4

	// adding path length and total path
	binary.BigEndian.PutUint32(buffer[offset:offset+4], uint32(8))
	offset += 4
	copy(buffer[offset:offset+8], e.path)
	offset += 8

	// adding name length and name bytes
	binary.BigEndian.PutUint32(buffer[offset:offset+4], uint32(len(name_data)))
	offset += 4
	copy(buffer[offset:offset+8], name_data)
	offset += len(name_data)

	// adding from length and from bytes
	binary.BigEndian.PutUint32(buffer[offset:offset+4], uint32(len(from_data)))
	offset += 4
	copy(buffer[offset:offset+8], from_data)
	offset += len(from_data)

	// adding target length and target bytes
	binary.BigEndian.PutUint32(buffer[offset:offset+4], uint32(len(target_data)))
	offset += 4
	copy(buffer[offset:offset+8], target_data)
	offset += len(target_data)

	// copying remaining Event Data
	copy(buffer[offset:], e.Data)

	return buffer
}

func (e *Event) FromBytes(buffer []byte) {
	path_skip_len := 4 + 8
	// skipping path information because in
	// API we don't need path information
	offset := path_skip_len

	// getting name length
	name_len := binary.BigEndian.Uint32(buffer[offset : offset+4])
	offset += 4
	// getting name filed
	e.Name = string(buffer[offset : offset+int(name_len)])
	offset += int(name_len)

	// getting From length
	from_len := binary.BigEndian.Uint32(buffer[offset : offset+4])
	offset += 4
	// getting From filed
	e.From = string(buffer[offset : offset+int(from_len)])
	offset += int(from_len)

	// getting Target length
	target_len := binary.BigEndian.Uint32(buffer[offset : offset+4])
	offset += 4
	// getting Target filed
	e.From = string(buffer[offset : offset+int(target_len)])
	offset += int(target_len)

	// copying remaining data as Event Data
	e.Data = make([]byte, len(buffer)-offset)
	copy(e.Data, buffer[offset:])
}
