package ethchain

import (
	"fmt"
	"math/big"
)

// Parent error. In case a parent is unknown this error will be thrown
// by the block manager
type ParentErr struct {
	Message string
}

func (err *ParentErr) Error() string {
	return err.Message
}

func ParentError(hash []byte) error {
	return &ParentErr{Message: fmt.Sprintf("Block's parent unkown %x", hash)}
}

func IsParentErr(err error) bool {
	_, ok := err.(*ParentErr)

	return ok
}

type UncleErr struct {
	Message string
}

func (err *UncleErr) Error() string {
	return err.Message
}

func UncleError(str string) error {
	return &UncleErr{Message: str}
}

func IsUncleErr(err error) bool {
	_, ok := err.(*UncleErr)

	return ok
}

// Block validation error. If any validation fails, this error will be thrown
type ValidationErr struct {
	Message string
}

func (err *ValidationErr) Error() string {
	return err.Message
}

func ValidationError(format string, v ...interface{}) *ValidationErr {
	return &ValidationErr{Message: fmt.Sprintf(format, v...)}
}

func IsValidationErr(err error) bool {
	_, ok := err.(*ValidationErr)

	return ok
}

type GasLimitErr struct {
	Message string
	Is, Max *big.Int
}

func IsGasLimitErr(err error) bool {
	_, ok := err.(*GasLimitErr)

	return ok
}
func (err *GasLimitErr) Error() string {
	return err.Message
}
func GasLimitError(is, max *big.Int) *GasLimitErr {
	return &GasLimitErr{Message: fmt.Sprintf("GasLimit error. Max %s, transaction would take it to %s", max, is), Is: is, Max: max}
}

type NonceErr struct {
	Message string
	Is, Exp uint64
}

func (err *NonceErr) Error() string {
	return err.Message
}

func NonceError(is, exp uint64) *NonceErr {
	return &NonceErr{Message: fmt.Sprintf("Nonce err. Is %d, expected %d", is, exp), Is: is, Exp: exp}
}

func IsNonceErr(err error) bool {
	_, ok := err.(*NonceErr)

	return ok
}

type OutOfGasErr struct {
	Message string
}

func OutOfGasError() *OutOfGasErr {
	return &OutOfGasErr{Message: "Out of gas"}
}
func (self *OutOfGasErr) Error() string {
	return self.Message
}

func IsOutOfGasErr(err error) bool {
	_, ok := err.(*OutOfGasErr)

	return ok
}


type InvalidPermErr struct{
    Message string
    Addr string
}

func InvalidPermError(addr, role string) *InvalidPermErr{
    return &InvalidPermErr{Message: fmt.Sprintf("Invalid permissions err on role %s for adddress %s", role, addr), Addr:addr}
}

func (self *InvalidPermErr) Error() string{
    return self.Message
}
