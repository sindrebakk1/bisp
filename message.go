package bisp

import (
	"errors"
	"fmt"
	"reflect"
)

const (
	VersionSize       = 1
	FlagsSize         = 1
	TypeIDSize        = 2
	TransactionIDSize = 16
	LengthSize        = 2
)

// Version is the version of the Message Header. Used to verify compatibility between client and server.
type Version uint8

const (
	V1 Version = 1
)

// CurrentVersion is the current Version of the protocol.
const CurrentVersion = V1

// Flag is a flag for the Message Header. Each flag is bit-shifted one position to the left, so a single byte can hold any combination of up to 8 flags.
type Flag uint8

const (
	// FError Flag is set if the message contains an error.
	FError Flag = 1 << iota
	// FTransaction Flag is set if the message contains a transaction ID.
	FTransaction Flag = 1 << 1
	// F32b Flag is set to use 32 bit lengths instead of 16 bit
	F32b Flag = 1 << 2
	// FHuff Flag is set if the message body is compressed using huffman encoding.
	FHuff Flag = 1 << 3
	// FProcedure Flag is set if the message is a Procedure call.
	FProcedure Flag = 1 << 4
)

// HeaderSize is the minimum size of the header in bytes.
const HeaderSize = VersionSize + FlagsSize + TypeIDSize + LengthSize

// MaxTcpMessageBodySize is the maximum size of a message in bytes
// max tcp packet size is 64KB, hence the subtraction of max header size, just to be safe
const MaxTcpMessageBodySize = 1<<16 - 1

const Max32bMessageBodySize = 1<<32 - 1

// ID is a unique identifier for a Message or Procedure type.
type ID uint16

// TransactionID is a unique identifier for a transaction.
type TransactionID [TransactionIDSize]byte

// Length is the length of the Message body.
type Length uint32

// Header is the header of a Message.
type Header struct {
	Version       Version
	Flags         Flag
	Type          ID
	TransactionID TransactionID
	Length        Length
}

// IsError returns true if the Header has the FError Flag set.
func (h *Header) IsError() bool {
	return h.Flags&FError == FError
}

// HasFlag returns true if the header has the Flag f.
func (h *Header) HasFlag(f Flag) bool {
	return h.Flags&f == f
}

// SetFlag sets the Flag f in the header.
func (h *Header) SetFlag(f Flag) {
	h.Flags |= f
}

// ClearFlag clears the Flag f in the header.
func (h *Header) ClearFlag(f Flag) {
	h.Flags &= ^f
}

// HasTransactionID returns true if the Header has a TransactionID. Will not update the header.
func (h *Header) HasTransactionID() bool {
	if h.HasFlag(FTransaction) {
		return true
	}
	hasTransactionID := false
	for _, b := range h.TransactionID {
		if b != 0 {
			hasTransactionID = true
			break
		}
	}
	return hasTransactionID
}

// Len returns the length of the Header in bytes.
func (h *Header) Len() int {
	l := HeaderSize
	if h.HasTransactionID() {
		l += TransactionIDSize
	}
	if h.HasFlag(F32b) {
		l += LengthSize
	}
	return l
}

// Message is a message sent between client and server.
type Message struct {
	Header Header
	Body   interface{}
}

// IsError returns true if the Message contains an error.
func (m *Message) IsError() bool {
	return m.Header.IsError()
}

// Error returns the error Message if the Message contains an error.
func (m *Message) Error() error {
	if !m.IsError() {
		return nil
	}
	errMsg, ok := m.Body.(string)
	if !ok {
		return errors.New(fmt.Sprintf("expected error body to be string, got %s", reflect.TypeOf(m.Body)))
	}
	return errors.New(errMsg)
}

// IsTransaction returns true if the Message contains a TransactionID.
func (m *Message) IsTransaction() bool {
	return m.Header.HasTransactionID()
}

// IsProcedure returns true if the Message is a Procedure call.
func (m *Message) IsProcedure() bool {
	return m.Header.HasFlag(FProcedure)
}

type TMessage[T any] struct {
	Header Header
	Body   T
}
