package service

import "testing"

func TestBase62Reversibility(t *testing.T) {
	encoder := NewBase62Encoder()

	testCases := []uint64{
		0,
		1,
		62,
		63,
		999999,
		18446744073709551615, // Max uint64
	}

	for _, tc := range testCases {
		encoded := encoder.Encode(tc)
		decoded, err := encoder.Decode(encoded)

		if err != nil {
			t.Fatalf("Base62 decode failed with error for input %d: %v", tc, err)
		}

		if decoded != tc {
			t.Errorf("Base62 failed for %d: Encoded to %s, Decoded to %d (Expected %d)", tc, encoded, decoded, tc)
		}
	}
}

func TestBase62InvalidCharacters(t *testing.T) {
	encoder := NewBase62Encoder()

	invalidStrings := []string{
		"abc-123",
		"xK9_4w",
		"3D space",
	}

	for _, tc := range invalidStrings {
		_, err := encoder.Decode(tc)
		if err == nil {
			t.Errorf("Expected decode failure for invalid string %q, but succeeded", tc)
		}
	}
}
