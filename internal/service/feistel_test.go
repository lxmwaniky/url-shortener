package service

import "testing"

func TestFeistelReversibility(t *testing.T) {
	feistel := NewFeistel(12345)

	testCases := []uint64{
		0,
		1,
		2,
		100,
		12345,
		99999999,
		18446744073709551615, // Max uint64
	}

	for _, tc := range testCases {
		encrypted := feistel.Encrypt(tc)
		decrypted := feistel.Decrypt(encrypted)

		if decrypted != tc {
			t.Errorf("Feistel failed for %d: Encrypted to %d, Decrypted to %d (Expected %d)", tc, encrypted, decrypted, tc)
		}
	}
}

func TestFeistelObfuscation(t *testing.T) {
	feistel := NewFeistel(987654)

	val1 := feistel.Encrypt(1)
	val2 := feistel.Encrypt(2)

	if val1 == val2 {
		t.Errorf("Feistel output is equal for different inputs: Encrypt(1) = %d, Encrypt(2) = %d", val1, val2)
	}

	// Ensure they don't look sequential
	diff := int64(val1) - int64(val2)
	if diff == 1 || diff == -1 {
		t.Errorf("Feistel outputs look sequential: Encrypt(1) = %d, Encrypt(2) = %d", val1, val2)
	}
}
