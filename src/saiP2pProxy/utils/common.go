package utils

import (
	"bytes"
	"errors"
)

func ExtractResult(input []byte) ([]byte, error) {
	_, after, found := bytes.Cut(input, []byte(":"))
	if !found {
		return nil, errors.New("wrong result!")
	}

	result := bytes.TrimSuffix(after, []byte("}"))
	return result, nil

}
