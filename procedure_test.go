package bisp_test

import (
	"bytes"
	"github.com/sindrebakk1/bisp"
	"testing"
)

func TestEncoder_EncodeProcedureCallBody(t *testing.T) {
	buf := new(bytes.Buffer)
	encoder := bisp.NewEncoder(buf)
	encoder.EncodeProcedureCallBody(procedure, 1, 2, 3, 4)
}

func TestEncoder_EncodeProcedureReturnBody(t *testing.T) {
	t.Skip("skipping test")
}

func TestDecoder_DecodeProcedureCallBody(t *testing.T) {
	t.Skip("skipping test")
}

func TestDecoder_DecodeProcedureReturnBody(t *testing.T) {
	t.Skip("skipping test")
}
