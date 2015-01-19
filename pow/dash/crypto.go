package dash

/*
char *sha3_cgo(char *, int); // Forward declaration
*/
import "C"
import (
	"github.com/eris-ltd/new-thelonious/crypto"
)

//export Sha3
func Sha3(data []byte, l int) []byte {
	return crypto.Sha3(data)
}
