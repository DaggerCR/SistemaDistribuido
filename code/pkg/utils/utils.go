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
		return nil, errors.New("impossible to assign task in an empty cluster")
	} else if chunkSize > arraySize {
		chunkSize = arraySize
	}

	// Reserve enough chunks for the main chunks
	amountChunks := arraySize / chunkSize
	var chunks [][]float64

	// Create chunks of size `chunkSize`
	for idx := 0; idx < amountChunks; idx++ {
		start := idx * chunkSize
		end := start + chunkSize
		chunk := make([]float64, chunkSize)
		copy(chunk, arrayToSum[start:end])
		chunks = append(chunks, chunk)
	}

	// Handle the remainder elements by creating a separate slice for each
	remainderStart := amountChunks * chunkSize
	for i := remainderStart; i < arraySize; i++ {
		chunk := []float64{arrayToSum[i]} // Create a single-element slice
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}
