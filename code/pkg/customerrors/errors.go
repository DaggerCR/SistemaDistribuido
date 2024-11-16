package customerrors

import (
	"errors"
	"fmt"
)

var CreationError = errors.New("error creando elemento")

func HandleError(e error) {
	fmt.Println("Error: ", e)
}
