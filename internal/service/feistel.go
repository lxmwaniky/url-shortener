package service

type Feistel struct {
	key uint32
}

func NewFeistel(key uint32) *Feistel {
	return &Feistel{key: key}
}

func (f *Feistel) Encrypt(val uint64) uint64 {
	left := uint32(val >> 32)
	right := uint32(val & 0xFFFFFFFF)

	for i := 0; i < 4; i++ {
		nextLeft := right
		scramble := (right ^ f.key ^ uint32(i)) * 2654435769
		nextRight := left ^ scramble
		left = nextLeft
		right = nextRight
	}

	return (uint64(left) << 32) | uint64(right)
}

func (f *Feistel) Decrypt(val uint64) uint64 {
	left := uint32(val >> 32)
	right := uint32(val & 0xFFFFFFFF)

	for i := 3; i >= 0; i-- {
		prevRight := left
		scramble := (left ^ f.key ^ uint32(i)) * 2654435769
		prevLeft := right ^ scramble
		left = prevLeft
		right = prevRight
	}

	return (uint64(left) << 32) | uint64(right)
}
