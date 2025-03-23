package aka

import (
	"5G_AKA/milenage"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

type Aka struct {
	// Milenage object
	mil milenage.Milenage

	// SNN
	SNN []byte

	// SUPI
	SUPI []byte

	KAUSF []byte
	KSEAF []byte
	KAMF  []byte

	HXRESStar []byte
}

func New(mil milenage.Milenage, SNN string, SUPI string) *Aka {
	a := &Aka{
		mil:   mil,
		SNN:   []byte(SNN),
		SUPI:  []byte(SUPI),
		KAUSF: make([]byte, 32),
		KSEAF: make([]byte, 32),
		KAMF:  make([]byte, 32),
	}

	return a
}

func (a *Aka) ComputeKAUSF() ([]byte, error) {
	sqnXorAk := milenage.Xor(a.mil.SQN, a.mil.AK)
	sqnXorAkLen := byteArrayLen2B(sqnXorAk)
	sNNLen := byteArrayLen2B(a.SNN)

	// Construct the input string
	inputString := []byte{0x6a}
	inputString = append(inputString, a.SNN...)
	inputString = append(inputString, sNNLen...)
	inputString = append(inputString, sqnXorAk...)
	inputString = append(inputString, sqnXorAkLen...)

	// Construct the input key
	inputKey := append(a.mil.CK, a.mil.IK...)

	// Compute HMAC-SHA256
	h := hmac.New(sha256.New, inputKey)
	h.Write(inputString)
	kausf := h.Sum(nil) // Get the hash result

	a.KAUSF = kausf
	return kausf, nil
}

func (a *Aka) ComputeKSEAF() ([]byte, error) {
	sNNLen := byteArrayLen2B(a.SNN)

	// Construct the input string
	inputString := []byte{0x6c}
	inputString = append(inputString, a.SNN...)
	inputString = append(inputString, sNNLen...)

	// Construct the input key
	inputKey := a.KAUSF

	// Compute HMAC-SHA256
	h := hmac.New(sha256.New, inputKey)
	h.Write(inputString)
	kseaf := h.Sum(nil) // Get the hash result

	a.KSEAF = kseaf
	return kseaf, nil
}

func (a *Aka) ComputeKAMF() ([]byte, error) {
	sUPILen := byteArrayLen2B(a.SUPI)

	// Construct the input string
	abba := []byte{0x00, 0x00}
	inputString := append([]byte{0x6d}, a.SUPI...)
	inputString = append(inputString, sUPILen...)
	inputString = append(inputString, abba...)
	inputString = append(inputString, []byte{0x00, 0x02}...)

	// Construct the input key
	inputKey := a.KAUSF

	// Compute HMAC-SHA256
	h := hmac.New(sha256.New, inputKey)
	h.Write(inputString)
	kamf := h.Sum(nil)

	a.KAMF = kamf
	return kamf, nil
}

func (a *Aka) ComputeHXRESStar() ([]byte, error) {
	// Construct the input string
	inputString := append(a.mil.RAND, a.mil.RESStar...)

	// Compute SHA256
	hash := sha256.Sum256(inputString)
	hxresstar := hash[len(hash)-16:]

	a.HXRESStar = hxresstar
	return hxresstar, nil
}

// takes in a byte array and
// returns a byte array of length 2 bytes containing the length of the input byte array
func byteArrayLen2B(b []byte) []byte {
	length := len(b)
	result := make([]byte, 2)
	binary.BigEndian.PutUint16(result, uint16(length))
	return result
}
