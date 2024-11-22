package customerrors

import (
	"distributed-system/logs"
	"errors"
	"fmt"
)

var CreationError = errors.New("error creando elemento")

func HandleError(e error) {
	logs.Initialize()
	logs.Log.Error("[ERROR] error", e)
	fmt.Println("Error, check log")
}
