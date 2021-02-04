package payment

import (
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	ethWallet "perun.network/go-perun/backend/ethereum/wallet"
	"perun.network/go-perun/channel"
	perunio "perun.network/go-perun/pkg/io"
	"perun.network/go-perun/wallet"
)

// Invoice serves as the data for the payment app.
type Invoice [32]byte

// App represents the payment app.
// It implements a channel.App and a channel.StateApp.
type App struct {
}

// Def returns the zero address.
func (a *App) Def() wallet.Address {
	return ethWallet.AsWalletAddr(common.Address{})
}

// DecodeData returns a decoded Invoice or an error.
func (a *App) DecodeData(r io.Reader) (channel.Data, error) {
	var data Invoice
	return &data, data.Decode(r)
}

// Decode decodes an Invoice from an io.Reader.
func (i *Invoice) Decode(r io.Reader) error {
	return perunio.Decode(r, (*[32]byte)(i))
}

// Encode encodes an Invoice into an io.Writer.
func (i Invoice) Encode(w io.Writer) error {
	return perunio.Encode(w, [32]byte(i))
}

// Clone returns a deep copy of an Invoice.
func (i Invoice) Clone() channel.Data {
	return &i
}

// ValidTransition checks that the data of the `to` state is of type Invoice.
func (a *App) ValidTransition(_ *channel.Params, _, to *channel.State, _ channel.Index) error {
	return assertInvoiceData(to.Data)
}

// ValidTransition checks that the data of the initial state is of type Invoice.
func (a *App) ValidInit(_ *channel.Params, state *channel.State) error {
	return assertInvoiceData(state.Data)
}

// assertInvoiceData asserts that the given data is of the type Invoice.
func assertInvoiceData(data channel.Data) error {
	if _, ok := data.(*Invoice); ok {
		return nil
	} else {
		return errors.Errorf("Invalid data type, must be Invoice, is %T", data)
	}
}
