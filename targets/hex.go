package targets

import (
	"encoding"
	"encoding/hex"
)

type Hex struct {
	Bytes []byte
}

var _ encoding.TextUnmarshaler = (*Hex)(nil)

func (h *Hex) UnmarshalText(text []byte) (err error) {
	h.Bytes, err = hex.DecodeString(string(text))
	return
}
