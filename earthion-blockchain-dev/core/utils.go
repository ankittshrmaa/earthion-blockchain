package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func IntToHex(n int64) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, n)
	return buf.Bytes()
}

// itoa converts int to string
func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}