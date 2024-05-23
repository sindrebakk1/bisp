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
	FError         Flag = 1 << iota
	FHuff          Flag = 1 << 1
	FTransactionID Flag = 1 << 2
)

const HeaderSize = VersionSize + FlagsSize + TypeIDSize + LengthSize

const HeaderSizeWithTransactionID = HeaderSize + TransactionIDSize

// MaxMessageBodySize is the maximum size of a message in bytes
// max tcp packet size is 64KB, hence the subtraction of max header size, just to be safe
const MaxMessageBodySize = 1<<16 - HeaderSizeWithTransactionID

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

type Message struct {
	Header Header
	Body   interface{}
}
