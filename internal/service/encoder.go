package service

import (
	"errors"
	"strings"
)

const (
	alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	base     = uint64(len(alphabet))
)

type Base62Encoder struct{}

func NewBase62Encoder() *Base62Encoder {
	return &Base62Encoder{}
}

func (e *Base62Encoder) Encode(num uint64) string {
	if num == 0 {
		return string(alphabet[0])
	}

	var builder strings.Builder
	for num > 0 {
		builder.WriteByte(alphabet[num%base])
		num /= base
	}

	bytes := []byte(builder.String())
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}

	return string(bytes)
}

func (e *Base62Encoder) Decode(str string) (uint64, error) {
	var result uint64
	for i := 0; i < len(str); i++ {
		char := str[i]
		idx := strings.IndexByte(alphabet, char)
		if idx == -1 {
			return 0, errors.New("invalid character in base62 string")
		}
		result = result*base + uint64(idx)
	}
	return result, nil
}
