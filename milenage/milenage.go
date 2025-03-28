// File downloaded from github.com/wmnsk/milenage (121f4a6)

// Copyright 2018-2023 milenage authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

/*
Package milenage provides the set of functions of MILENAGE algorithm set defined in 3GPP TS 35.205
and some helpers to be used during the authentication procedure.
*/
package milenage

import (
	"crypto/aes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// Milenage is a set of parameters used/generated in MILENAGE algorithm.
type Milenage struct {
	// K is a 128-bit subscriber key that is an input to the functions f1, f1*, f2, f3, f4, f5 and f5*.
	K []byte
	// OP is a 128-bit Operator Variant Algorithm Configuration Field that is a component of the
	// functions f1, f1*, f2, f3, f4, f5 and f5*.
	OP []byte
	// OPc is a 128-bit value derived from OP and K and used within the computation of the functions.
	OPc []byte
	// RAND is a 128-bit random challenge that is an input to the functions f1, f1*, f2, f3, f4, f5 and f5*.
	RAND []byte

	// SQN is a 48-bit sequence number that is an input to either of the functions f1 and f1*.
	// (For f1* this input is more precisely called SQNMS.)
	SQN []byte
	// AMF is a 16-bit authentication management field that is an input to the functions f1 and f1*.
	AMF []byte

	// MACA is a 64-bit network authentication code that is the output of the function f1.
	MACA []byte
	// MACS is a 64-bit resynchronisation authentication code that is the output of the function f1*.
	MACS []byte

	// RES is a 64-bit signed response that is the output of the function f2.
	RES []byte
	// CK is a 128-bit confidentiality key that is the output of the function f3.
	CK []byte
	// IK is a 128-bit integrity key that is the output of the function f4.
	IK []byte
	// AK is a 48-bit anonymity key that is the output of either of the functions f5.
	AK []byte
	// AKS is a 48-bit anonymity key that is the output of either of the functions f5*.
	AKS []byte

	// RESStar or RES* is a 128-bit response that is used in 5G.
	RESStar []byte
}

// New initializes a new MILENAGE algorithm.
func New(k, op, rand []byte, sqn uint64, amf uint16) *Milenage {
	m := &Milenage{
		K:    k,
		OP:   op,
		OPc:  nil,
		RAND: rand,
		AMF:  make([]byte, 2),
		SQN:  make([]byte, 6),
		MACA: make([]byte, 8),
		MACS: make([]byte, 8),
		RES:  make([]byte, 8),
		CK:   make([]byte, 16),
		IK:   make([]byte, 16),
		AK:   make([]byte, 6),
		AKS:  make([]byte, 6),
	}

	s := make([]byte, 8)
	binary.BigEndian.PutUint64(s, sqn)
	for i := 0; i < 6; i++ {
		m.SQN[i] = s[i+2]
	}

	binary.BigEndian.PutUint16(m.AMF, amf)

	return m
}

// NewWithOPc initializes a new MILENAGE algorithm using OPc instead of OP.
func NewWithOPc(k, opc, rand []byte, sqn uint64, amf uint16) *Milenage {
	m := &Milenage{
		K:    k,
		OP:   nil,
		OPc:  opc,
		RAND: rand,
		AMF:  make([]byte, 2),
		SQN:  make([]byte, 6),
		MACA: make([]byte, 8),
		MACS: make([]byte, 8),
		RES:  make([]byte, 8),
		CK:   make([]byte, 16),
		IK:   make([]byte, 16),
		AK:   make([]byte, 6),
		AKS:  make([]byte, 6),
	}

	s := make([]byte, 8)
	binary.BigEndian.PutUint64(s, sqn)
	for i := 0; i < 6; i++ {
		m.SQN[i] = s[i+2]
	}

	binary.BigEndian.PutUint16(m.AMF, amf)

	return m
}

// ComputeOPc is a helper that provides users to retrieve OPc value from
// the K and OP given.
func ComputeOPc(k, op []byte) ([]byte, error) {
	m := New(k, op, make([]byte, 16), 0, 0)
	if err := m.computeOPc(); err != nil {
		return nil, err
	}
	return m.OPc, nil
}

// ComputeAll fills all the fields in *Milenage struct.
func (m *Milenage) ComputeAll() error {
	if err := m.validateLength(); err != nil {
		return err
	}

	if _, err := m.F1(); err != nil {
		return fmt.Errorf("F1() failed: %w", err)
	}

	if _, err := m.F1Star(m.SQN, m.AMF); err != nil {
		return fmt.Errorf("F1Star() failed: %w", err)
	}

	if _, _, _, _, err := m.F2345(); err != nil {
		return fmt.Errorf("F2345() failed: %w", err)
	}

	if _, err := m.F5Star(); err != nil {
		return fmt.Errorf("F5Star() failed: %w", err)
	}

	return nil
}

// F1 is the network authentication function.
// F1 computes network authentication code MAC-A from key K, random challenge RAND,
// sequence number SQN and authentication management field AMF.
func (m *Milenage) F1() ([]byte, error) {
	mac, err := m.f1base(m.SQN, m.AMF)
	if err != nil {
		return nil, err
	}

	m.MACA = mac[:8]
	return mac[:8], nil
}

// F1Star is the re-synchronisation message authentication function.
// F1Star computes resynch authentication code MAC-S from key K, random challenge RAND,
// sequence number SQN and authentication management field AMF.
//
// Note that the AMF value should be zero to be compliant with the specification
// TS 33.102 6.3.3 (This method just computes with the given value).
func (m *Milenage) F1Star(sqn, amf []byte) ([]byte, error) {
	mac, err := m.f1base(sqn, amf)
	if err != nil {
		return nil, err
	}

	m.MACS = mac[8:]
	return mac[8:], nil
}

// F2345 takes key K and random challenge RAND, and returns response RES,
// confidentiality key CK, integrity key IK and anonymity key AK.
func (m *Milenage) F2345() (res, ck, ik, ak []byte, err error) {
	if err := m.validateLength(); err != nil {
		return nil, nil, nil, nil, err
	}

	if m.OPc == nil {
		if err := m.computeOPc(); err != nil {
			return nil, nil, nil, nil, err
		}
	}

	rijndaelInput := make([]byte, 16)
	for i := 0; i < 16; i++ {
		rijndaelInput[i] = m.RAND[i] ^ m.OPc[i]
	}

	temp, err := encrypt(m.K, rijndaelInput)
	if err != nil {
		return
	}

	// To obtain output block OUT2: XOR OPc and TEMP, rotate by r2=0, and XOR on the
	// constant c2 (which is all zeroes except that the last bit is 1).
	for i := 0; i < 16; i++ {
		rijndaelInput[i] = temp[i] ^ m.OPc[i]
	}
	rijndaelInput[15] ^= 1

	out, err := encrypt(m.K, rijndaelInput)
	if err != nil {
		return
	}
	tmp := xor(out, m.OPc)
	res = tmp[8:]
	ak = tmp[:6]

	// To obtain output block OUT3: XOR OPc and TEMP, rotate by r3=32, and XOR on the
	// constant c3 (which is all zeroes except that the next to last bit is 1).
	for i := 0; i < 16; i++ {
		rijndaelInput[(i+12)%16] = temp[i] ^ m.OPc[i]
	}
	rijndaelInput[15] ^= 2

	out, err = encrypt(m.K, rijndaelInput)
	if err != nil {
		return
	}
	ck = xor(out, m.OPc)

	// To obtain output block OUT4: XOR OPc and TEMP, rotate by r4=64, and XOR on the
	// constant c4 (which is all zeroes except that the 2nd from last bit is 1).

	for i := 0; i < 16; i++ {
		rijndaelInput[(i+8)%16] = temp[i] ^ m.OPc[i]
	}
	rijndaelInput[15] ^= 4

	out, err = encrypt(m.K, rijndaelInput)
	if err != nil {
		return
	}
	ik = xor(out, m.OPc)

	m.RES = res
	m.CK = ck
	m.IK = ik
	m.AK = ak
	return res, ck, ik, ak, nil
}

// F5Star is the anonymity key derivation function for the re-synchronisation message.
// F5Star takes key K and random challenge RAND, and returns resynch anonymity key AK.
func (m *Milenage) F5Star() (aks []byte, err error) {
	if err := m.validateLength(); err != nil {
		return nil, err
	}

	if m.OPc == nil {
		if err := m.computeOPc(); err != nil {
			return nil, err
		}
	}

	rijndaelInput := make([]byte, 16)
	for i := 0; i < 16; i++ {
		rijndaelInput[i] = m.RAND[i] ^ m.OPc[i]
	}

	tmp, err := encrypt(m.K, rijndaelInput)
	if err != nil {
		return
	}

	// To obtain output block OUT5: XOR OPc and TEMP, rotate by r5=96, and XOR on the
	// constant c5 (which is all zeroes except that the 3rd from last bit is 1).
	for i := 0; i < 16; i++ {
		rijndaelInput[(i+4)%16] = tmp[i] ^ m.OPc[i]
	}
	rijndaelInput[15] ^= 8

	out, err := encrypt(m.K, rijndaelInput)
	if err != nil {
		return
	}

	aks = xor(out, m.OPc)[:6]
	m.AKS = aks
	return aks, nil
}

// ComputeRESStar computes RESStar from serving network name, RAND and RES
// as described in A.4 RES* and XRES* derivation function, TS 33.501.
//
// Note that this function should be called after all other calculations
// is done (to generate RAND and RES).
func (m *Milenage) ComputeRESStar(mcc, mnc string) ([]byte, error) {
	if err := m.validateLength(); err != nil {
		return nil, err
	}

	if len(mcc) != 3 {
		return nil, fmt.Errorf("invalid MCC: %s", mcc)
	}
	if l := len(mnc); l == 2 {
		mnc = "0" + mnc
	} else if l != 3 {
		return nil, fmt.Errorf("invalid MNC: %s", mnc)
	}

	snn := []byte(fmt.Sprintf("5G:mnc%s.mcc%s.3gppnetwork.org", mnc, mcc))
	if l := len(snn); l != 32 {
		return nil, fmt.Errorf("failed to build SNN: %s", snn)
	}

	b := make([]byte, 63)
	b[0] = 0x6b

	copy(b[1:33], snn)
	binary.BigEndian.PutUint16(b[33:35], uint16(len(snn)))

	copy(b[35:51], m.RAND)
	binary.BigEndian.PutUint16(b[51:53], uint16(len(m.RAND)))

	copy(b[53:61], m.RES)
	binary.BigEndian.PutUint16(b[61:63], uint16(len(m.RES)))

	k := make([]byte, 32)
	copy(k[0:16], m.CK)
	copy(k[16:32], m.IK)
	mac := hmac.New(sha256.New, k)
	if _, err := mac.Write(b); err != nil {
		return nil, fmt.Errorf("failed to compute RES*: %w", err)
	}

	out := mac.Sum(nil)
	return out[len(out)-16:], nil
}

// GenerateAUTN generates AUTN uing the current values in Milenage
// in the way described in 5.1.1.1, TS 33.105 and 6.3.2, TS 33.102.
func (m *Milenage) GenerateAUTN() ([]byte, error) {
	if err := m.validateLength(); err != nil {
		return nil, err
	}

	autn := make([]byte, 16)
	copy(autn[0:6], xor(m.SQN, m.AK))
	copy(autn[6:8], m.AMF)
	copy(autn[8:16], m.MACA)
	return autn, nil
}

// GenerateAUTS generates AUTS using the current values in Milenage
// in the way described in 5.1.1.3, TS 33.105 and 6.3.3, TS 33.102.
//
// Note: MAC-S and AK-S are re-calculated with AMF=0x0000.
func (m *Milenage) GenerateAUTS() ([]byte, error) {
	if err := m.validateLength(); err != nil {
		return nil, err
	}

	// The AMF used to calculate MAC-S assumes a dummy value of all
	// zeros so that it does not need to be transmitted in the clear
	// in the re-synch message (6.3.3, TS 33.102).
	macS, err := m.F1Star(m.SQN, []byte{0x00, 0x00})
	if err != nil {
		return nil, err
	}
	aks, err := m.F5Star()
	if err != nil {
		return nil, err
	}

	auts := make([]byte, 14)
	copy(auts[0:6], xor(m.SQN, aks))
	copy(auts[6:14], macS)

	return auts, nil
}

// computeOPc computes OPc from K and OP inside m.
func (m *Milenage) computeOPc() error {
	m.OPc = make([]byte, 16)

	block, err := aes.NewCipher(m.K)
	if err != nil {
		return err
	}
	cipherText := make([]byte, len(m.OP))
	block.Encrypt(cipherText, m.OP)

	bytes := xor(cipherText, m.OP)
	for i, b := range bytes {
		if i > len(m.OPc) {
			break
		}
		m.OPc[i] = b
	}
	return nil
}

func xor(b1, b2 []byte) []byte {
	var l int
	if len(b1)-len(b2) < 0 {
		l = len(b1)
	} else {
		l = len(b2)
	}

	// don't update b1
	out := make([]byte, l)
	for i := 0; i < l; i++ {
		out[i] = b1[i] ^ b2[i]
	}
	return out
}

func encrypt(key, plain []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	encrypted := make([]byte, len(plain))
	block.Encrypt(encrypted, plain)
	return encrypted, nil
}

func (m *Milenage) f1base(sqn, amf []byte) ([]byte, error) {
	if err := m.validateLength(); err != nil {
		return nil, err
	}

	if m.OPc == nil {
		if err := m.computeOPc(); err != nil {
			return nil, err
		}
	}

	rijndaelInput := make([]byte, 16)
	for i := 0; i < 16; i++ {
		rijndaelInput[i] = m.RAND[i] ^ m.OPc[i]
	}

	temp, err := encrypt(m.K, rijndaelInput)
	if err != nil {
		return nil, err
	}

	in1 := make([]byte, 16)
	for i := 0; i < 6; i++ {
		in1[i] = sqn[i]
		in1[i+8] = sqn[i]
	}
	for i := 0; i < 2; i++ {
		in1[i+6] = amf[i]
		in1[i+14] = amf[i]
	}

	// XOR op_c and in1, rotate by r1=64, and XOR
	// on the constant c1 (which is all zeroes)
	for i := 0; i < 16; i++ {
		rijndaelInput[(i+8)%16] = in1[i] ^ m.OPc[i]
	}
	/* XOR on the value temp computed before */

	for i := 0; i < 16; i++ {
		rijndaelInput[i] ^= temp[i]
	}

	out, err := encrypt(m.K, rijndaelInput)
	if err != nil {
		return nil, err
	}

	return xor(out, m.OPc), nil
}

func (m *Milenage) validateLength() error {
	if len(m.K) != 16 {
		return fmt.Errorf("length of K should be %d, got: %d", 16, len(m.K))
	}
	if m.OP != nil && len(m.OP) != 16 {
		return fmt.Errorf("length of OP should be %d, got: %d", 16, len(m.OP))
	}
	if m.OPc != nil && len(m.OPc) != 16 {
		return fmt.Errorf("length of OPc should be %d, got: %d", 16, len(m.OPc))
	}
	if len(m.RAND) != 16 {
		return fmt.Errorf("length of RAND should be %d, got: %d", 16, len(m.RAND))
	}
	if len(m.SQN) != 6 {
		return fmt.Errorf("length of SQN should be %d, got: %d", 6, len(m.SQN))
	}
	if len(m.AMF) != 2 {
		return fmt.Errorf("length of AMF should be %d, got: %d", 2, len(m.AMF))
	}
	if len(m.MACA) != 8 {
		return fmt.Errorf("length of MACA should be %d, got: %d", 8, len(m.MACA))
	}
	if len(m.MACS) != 8 {
		return fmt.Errorf("length of MACS should be %d, got: %d", 8, len(m.MACS))
	}
	if len(m.RES) != 8 {
		return fmt.Errorf("length of RES should be %d, got: %d", 8, len(m.RES))
	}
	if len(m.CK) != 16 {
		return fmt.Errorf("length of CK should be %d, got: %d", 16, len(m.CK))
	}
	if len(m.IK) != 16 {
		return fmt.Errorf("length of IK should be %d, got: %d", 16, len(m.IK))
	}
	if len(m.AK) != 6 {
		return fmt.Errorf("length of AK should be %d, got: %d", 6, len(m.AK))
	}
	if len(m.AKS) != 6 {
		return fmt.Errorf("length of AKS should be %d, got: %d", 6, len(m.AKS))
	}

	return nil
}

// DisplayMilenage prints all fields of a Milenage struct
func (m *Milenage) DisplayMilenage() {
	fmt.Println("Milenage Struct Contents:")
	fmt.Println("K       :", hex.EncodeToString(m.K))
	fmt.Println("OP      :", hex.EncodeToString(m.OP))
	fmt.Println("OPc     :", hex.EncodeToString(m.OPc))
	fmt.Println("RAND    :", hex.EncodeToString(m.RAND))
	fmt.Println("SQN     :", hex.EncodeToString(m.SQN))
	fmt.Println("AMF     :", hex.EncodeToString(m.AMF))
	fmt.Println("MACA    :", hex.EncodeToString(m.MACA))
	fmt.Println("MACS    :", hex.EncodeToString(m.MACS))
	fmt.Println("RES     :", hex.EncodeToString(m.RES))
	fmt.Println("CK      :", hex.EncodeToString(m.CK))
	fmt.Println("IK      :", hex.EncodeToString(m.IK))
	fmt.Println("AK      :", hex.EncodeToString(m.AK))
	fmt.Println("AKS     :", hex.EncodeToString(m.AKS))
	fmt.Println("RESStar :", hex.EncodeToString(m.RESStar))
}

func Xor(b1, b2 []byte) []byte {
	return xor(b1, b2)
}
