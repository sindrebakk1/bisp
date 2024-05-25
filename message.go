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
	// FHuff Flag is set if the message body is compressed using huffman encoding.
	FHuff Flag = 1 << 1
	// FTransactionID Flag is set if the message contains a transaction ID.
	FTransactionID Flag = 1 << 2
	// FBigLengths Flag is set to use 32 bit lengths instead of 16 bit for strings, slices and maps.
	FBigLengths Flag = 1 << 3
)

const HeaderSize = VersionSize + FlagsSize + TypeIDSize + LengthSize

const HeaderSizeWithTransactionID = HeaderSize + TransactionIDSize

// MaxTcpMessageBodySize is the maximum size of a message in bytes
// max tcp packet size is 64KB, hence the subtraction of max header size, just to be safe
const MaxTcpMessageBodySize = 1<<16 - HeaderSizeWithTransactionID

type TypeID uint16

type TransactionID [TransactionIDSize]byte

type Length uint16

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

type Message struct {
	Header Header
	Body   interface{}
}
