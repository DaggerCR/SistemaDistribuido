package errors

import (
	"errors"
	"fmt"
)

var CreationError = errors.New("error creando elemento")

func HandleError(e error) {
	switch e {
	case CreationError:
	}
	fmt.Println("Error: ", e)
}
