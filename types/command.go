package types

import (
	"encoding/json"
	"io"

	"github.com/google/uuid"
)

// CommandID is unique identifier for type of command for entire system.
type CommandID string

// Command is an interface for all commands for managing encoding and decoding
// for sending over the network. Concrete command parameters can be stored as a
// payload.
type Command interface {
	// ID returns unique identifier of this command, usually something like "user.create"
	ID() CommandID

	// CorrelationID returns unique identifier for this instance of the command, so that
	// it's effects can be traced across the system, usually randomly generated string
	CorrelationID() string

	// StorePayload stores value provided within a command as a payload.
	StorePayload(interface{}) error

	// LoadPayload populates provided interface (has to be a pointer) from
	// stored payload.
	LoadPayload(interface{}) error

	// Marshal returns byte slice representation of a command, suitable for
	// sending over the wire.
	Marshal() ([]byte, error)

	// Unmarshal recreates command from byte slice representation, useful for
	// reconstruction after sending over the wire.
	Unmarshal([]byte) error
}

type BaseCmd struct {
	// fields have to be public in order to be accessible to JSON serialization

	Xid            CommandID `json:"id,omitempty"`
	XcorrelationID string    `json:"correlation_id,omitempty"`
	Xpayload       []byte    `json:"payload,omitempty"`
}

func (c *BaseCmd) ID() CommandID               { return c.Xid }
func (c *BaseCmd) CorrelationID() string       { return c.XcorrelationID }
func (c *BaseCmd) Marshal() ([]byte, error)    { return json.Marshal(c) }
func (c *BaseCmd) Unmarshal(data []byte) error { return json.Unmarshal(data, c) }

func (c *BaseCmd) StorePayload(payload interface{}) error {
	d, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	c.Xpayload = d
	return nil
}

func (c *BaseCmd) LoadPayload(to interface{}) error {
	return json.Unmarshal(c.Xpayload, to)
}

// NewCommand returns command instance with provided ID and payload.
func NewCommand(ID CommandID, payload interface{}) (Command, error) {
	c := &BaseCmd{
		Xid:            ID,
		XcorrelationID: uuid.NewString(),
	}
	return c, c.StorePayload(payload)
}

// UnmarshalCommand creates and populates new command instance from provided raw data.
func UnmarshalCommand(raw []byte) (Command, error) {
	c := new(BaseCmd)
	return c, c.Unmarshal(raw)
}

// UnmarshalCommandFromReader creates and populates new command instance
// with data from provided reader. Note that max 10 MB of data will be read in
// order to prevent memory leaks. If you have a need for larger payloads than
// that (since it is mostly payload that contributes to Command size), custom
// implementation is needed.
func UnmarshalCommandFromReader(in io.Reader) (Command, error) {
	// reader might come from network connection (e.g. from *http.Request.Body)
	// so limit its size in oder to prevent taking too much memory
	// 10 MB should be reasonable
	dataReader := io.LimitReader(in, 10*1024*1024)
	raw, err := io.ReadAll(dataReader)
	if err != nil {
		return nil, err
	}
	return UnmarshalCommand(raw)
}

var _ Command = &BaseCmd{}
