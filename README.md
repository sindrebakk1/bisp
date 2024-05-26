# __Bi__*(nary)*__S__*(erialization)*__P__*(rotocol)*

__BiSP__ is a binary serialization protocol that is designed to be simple, fast, and efficient. It is designed to be used
in situations where JSON is too slow, and Protobuf is too complex. The protocol itself is language agnostic, but the go
implementation relies on reflection to encode and decode arbitrary data. Implementations in other languages will need to use
their own reflection libraries to achieve the same functionality, and for interop between language care has to be taken
so corresponding types and IDs are synced between the sender and receiver.

## How it works
The protocol uses a header to contain information about the type and length of the payload, as well as flags that can be
set to enable extra features, (like compression, transactionID etc.) The payload is then serialized using the 16 bit type ID.
For this to work, the types that are to be serialized have to be registered with the bisp package. Primitive types are
automatically registered, but structs, arrays, and maps, as well as any type aliases, have to be registered manually.
Registering types should be done during the init phase of the program, and has to be done in the same order on the server
and client. The library does expose a method to sync the type IDs from an external source, but receiving the
type registry map from the server has to be implemented manually.

## Example

```go
package main

import (
	"fmt"
	"github.com/sindrebakk1/bisp"
	"net"
)

type TestStruct struct {
    Bool   bool
    Int    int
    Float  float64
    String string
}

func main() {
	msg := bisp.Message{
		Body: TestStruct{
			Bool:   true,
			Int:    42,
			Float:  3.14,
			String: "Hello, World!",
		},
	}

	client, server := net.Pipe()

	go func() {
		encoder := bisp.NewEncoder(server)
        err := encoder.Encode(msg)
        if err != nil {
            panic(err)
        }
    }()
    decoder := bisp.NewDecoder(client)
	var response bisp.Message
	err := decoder.Decode(&response)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", response.Body.(TestStruct))
}

func init() {
	// Register the struct with the bisp package
	// This is necessary for the encoder and decoder to be able to encode and decode the struct
	// Types that have to be registered are; Structs, Arrays, and Maps as well as any type aliases such as enums
	bisp.RegisterType(TestStruct{})
}
```

## Protocol
> ### Header
> - version: 1 byte - the version of the protocol
> - flags: 1 byte - flags that can be set to enable extra features
> - type: 2 bytes - the type ID of the payload
> - transaction ID: 16 bytes - optional, only present if the transaction flag is set
> - payload length: 2 bytes - the length of the payload
> ### Payload
> - payload: anything that fits in a tcp packet

## Flags
> - FError: Error - If this flag is set, the payload is an error message
> - FHuff: Huffman - If this flag is set, the payload is compressed using the huffman algorithm
> - FTransaction: Transaction - If this flag is set, the transaction id is present in the header
> - FBigLen: Big Length - If this flag is set, the lengths for strings, slices and maps is 4 bytes instead of 2

## Primitive Types
TODO

## TODO
- [ ] Implement a way to sync the type registry
- [ ] Implement huffman encoding and decoding
- [ ] Implement a way to encrypt the payload?
- [ ] Tests:
  - [ ] Arrays
  - [ ] two-dimensional slices and arrays
- [ ] ~~RPC(like) implementation? Actions? Commands? Will require breaking changes to the protocol~~
