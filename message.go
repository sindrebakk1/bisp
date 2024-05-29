package bisp

const (
	VersionSize       = 1
	FlagsSize         = 1
	TypeIDSize        = 2
	TransactionIDSize = 16
	LengthSize        = 2
)

type Version uint8

const (
	V1 Version = 1
)

const CurrentVersion = V1

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
)

const HeaderSize = VersionSize + FlagsSize + TypeIDSize + LengthSize

const HeaderSizeWithTransactionID = HeaderSize + TransactionIDSize

// MaxTcpMessageBodySize is the maximum size of a message in bytes
// max tcp packet size is 64KB, hence the subtraction of max header size, just to be safe
const MaxTcpMessageBodySize = 1<<16 - 1

const Max32bMessageBodySize = 1<<32 - 1

type TypeID uint16

type TransactionID [TransactionIDSize]byte

type Length uint32

type Header struct {
	Version       Version
	Flags         Flag
	Type          TypeID
	TransactionID TransactionID
	Length        Length
}

func (h *Header) IsError() bool {
	return h.Flags&FError == FError
}

func (h *Header) HasFlag(f Flag) bool {
	return h.Flags&f == f
}

func (h *Header) SetFlag(f Flag) {
	h.Flags |= f
}

func (h *Header) ClearFlag(f Flag) {
	h.Flags &= ^f
}

func (h *Header) Len() int {
	l := HeaderSize
	if h.HasFlag(FTransaction) {
		l += TransactionIDSize
	}
	if h.HasFlag(F32b) {
		l += LengthSize
	}
	return l
}

type Message struct {
	Header Header
	Body   interface{}
}

type TMessage[T any] struct {
	Header Header
	Body   T
}
