package utils

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/joho/godotenv"
)

func LoadVEnv() error {
	if err := godotenv.Load(); err != nil {
		return errors.New("internal server error")
	}
	fmt.Println("Loading venv:sucess")
	return nil
}

func ParseStringUint8(s string) (uint, error) {
	num, errparse := strconv.ParseUint(s, 10, 8)
	if errparse != nil {
		return 0, errors.New("internal sever error")
	}
	return uint(num), nil
}

func SafeAccess[T int64 | float64](slice []T, index int) (T, bool) {
	if index >= 0 && index < len(slice) {
		return slice[index], true
	}
	return 0, false
}
