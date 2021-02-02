// ...

package prnm

import (
	"bufio"
)

// ClosableBufio ..-
type ClosableBufio struct {
	//bytes.Buffer
	bufio.Reader
	bufio.Writer
}

// Close ...
func (cb *ClosableBufio) Close() error {
	//cb.Buffer.Reset()
	return nil
}
