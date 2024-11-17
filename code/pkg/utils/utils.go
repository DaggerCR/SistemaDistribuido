package utils

import (
	"bufio"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Types
type NodeId int
type Load int
type AccumulatedChecks int

// functions
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

func GenRandomUint() uint {
	src := uint64(time.Now().UnixNano())
	r := rand.New(rand.NewPCG(1, src))
	return (r.UintN(10000))
}

func ReadEntry() string {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text()
}

func SliceUpArray(arrayToSum []float64, chunkSize int) ([][]float64, error) {
	arraySize := len(arrayToSum)
	if chunkSize <= 0 {
		return nil, errors.New("chunk size must be greater than 0")
	} else if chunkSize > arraySize {
		chunkSize = arraySize
	}

	var chunks [][]float64
	for start := 0; start < arraySize; start += chunkSize {
		end := start + chunkSize
		if end > arraySize {
			end = arraySize
		}
		chunks = append(chunks, arrayToSum[start:end])
	}

	return chunks, nil
}
