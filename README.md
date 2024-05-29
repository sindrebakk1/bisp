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
	// Send a message
	go func() {
		encoder := bisp.NewEncoder(server)
		err := encoder.Encode(msg)
		if err != nil {
			panic(err)
		}
	}()
	
	// Receive a message
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
> ### Header (6 <> 24 bytes)
> - version: 1 byte - the version of the protocol
> - flags: 1 byte - flags that can be set to enable extra features
> - type: 2 bytes - the type ID of the payload
> - transaction ID: 16 | 0 bytes - only present if the FTransaction flag is set
> - payload length: 2 | 4 bytes - the length of the payload, 4 bytes if the F32b flag is set
> ### Payload (0 <> 2^16 | 2^32 bytes)
> - payload: 0 - 2^16 | 2^32 bytes - the serialized payload

## Flags
> FError: Error - If this flag is set, the payload is an error message
> FTransaction: Transaction - If this flag is set, the transaction id is present in the header
> F32b: 32 bit lengths - If this flag is set, all lengths are 32 bits instead of 16 bits
> FHuff: Huffman - If this flag is set, the payload will be compressed using the huffman algorithm
> FRle: Run Length Encoding - If this flag is set, the payload will be compressed using the Run Length Encoding algorithm
> FEnc: Encryption - If this flag is set, the payload will be encrypted
> FProc: Procedure call - If this flag is set, the payload is a procedure call

## Primitive Types
TODO

## Procedure Calls
TODO

## TODO
- [ ] Features:
  - [X] Transaction ID
  - [X] 32 bit lengths
  - [ ] Register types by name to avoid having to register them in the same order
  - [ ] Procedure calls
  - [ ] Error handling - only relevant for procedures?
  - [ ] Compression
  - [ ] Encryption
  - [ ] Type syncing
- [ ] Tests:
  - [x] Arrays
  - [ ] Error handling
  - [ ] Type aliases
  - [ ] Two-dimensional slices and arrays
- [ ] Benchmarks
  - [X] Encoding
  - [X] Decoding
  - [X] Receive and Respond
  - [X] Large messages
    - [X] 16b length
    - [X] 32b length
  - [ ] Compression
- [ ] Optimizations
  - [ ] Use a pool for the encoder and decoder?
  - [ ] Use a pool for the huffman encoder and decoder?
  - [ ] Use unsafe pointers to avoid reflection where reasonable
  - [X] Large string, slice and array optimizations
  - [ ] Large struct optimizations
  - [ ] Large map optimizations
- [ ] Documentation
  - [ ] Examples
    - [X] Simple
    - [ ] Complex
    - [ ] Error handling
    - [ ] Compression
    - [ ] Transaction ID
    - [ ] Encryption
    - [ ] Procedure calls
  - [X] Protocol
  - [X] Flags
  - [ ] Procedure calls
  - [ ] Primitive types
  - [ ] Type registration
  - [ ] Type syncing
