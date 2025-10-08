package tools

import (
	"log"
	"os"
	"strconv"
)

type EnvParserFunc[T any] func(string) (T, error)

func GetenvOrDefault[T any](key string, defaultValue T, parseFn EnvParserFunc[T]) T {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}

	res, err := parseFn(val)
	if err != nil {
		log.Println(err)
		return defaultValue
	}
	return res
}

func IntParse(s string) (int, error) {
	return strconv.Atoi(s)
}
func StringParse(s string) (string, error) {
	return s, nil
}
