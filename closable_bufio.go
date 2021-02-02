// ...

package prnm

import (
	"bufio"
)

// ClosableBufio ..-
type ClosableBufio struct {
	bufio.ReadWriter
}

// Close ...
func (cb *ClosableBufio) Close() error {
	return nil
}
