package main

import (
	// crand "crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"strconv"

	"5G_AKA/aka"
	"5G_AKA/milenage"
)

func main() {
	var (
		imsis = flag.String("imsi", "001010123456789", "IMSI in string") // supported length: MCC 3 MNC 2
		ks    = flag.String("k", "00112233445566778899aabbccddeeff", "K in hex string")
		ops   = flag.String("op", "00112233445566778899aabbccddeeff", "OP in hex string")
		sqns  = flag.String("sqn", "000000000001", "SQN in hex string")
		amfs  = flag.String("amf", "8000", "AMF in hex string")
		rands = flag.String("rand", "00112233445566778899aabbccddeeff", "RAND in hex string")
	)
	flag.Parse()

	// provided by UDM
	k, err := hex.DecodeString(*ks)
	if err != nil {
		log.Fatalf("Invalid K \"%s\": %+v", *ks, err)
	}

	// provided by UDM
	op, err := hex.DecodeString(*ops)
	if err != nil {
		log.Fatalf("Invalid OP \"%s\": %+v", *ops, err)
	}
	opc, err := milenage.ComputeOPc(k, op)
	if err != nil {
		log.Fatalf("Failed to compute OPc: %+v", err)
	}

	// provided by UDM
	sqn, err := strconv.ParseUint(*sqns, 16, 64)
	if err != nil {
		log.Fatalf("Invalid SQN \"%s\": %+v", *sqns, err)
	}

	amf64, err := strconv.ParseUint(*amfs, 16, 16)
	amf := uint16(amf64)
	if err != nil {
		log.Fatalf("Invalid AMF \"%s\": %+v", *amfs, err)
	}

	// RAND from CLI
	rand, err := hex.DecodeString(*rands)
	if err != nil {
		log.Fatalf("Invalid RAND \"%s\": %+v", *rands, err)
	}
	// RAND from random
	// rand = make([]byte, 16)
	// _, err = crand.Read(rand)
	// if err != nil {
	// 	log.Fatalf("Failed to generate random RAND: %+v", err)
	// }

	mcc := (*imsis)[0:3]
	mnc := (*imsis)[3:5]

	fmt.Printf("IMSI     = %s %s %s\n", mcc, mnc, (*imsis)[5:])
	fmt.Printf("K        = %x\n", k)
	fmt.Printf("OPc      = %x\n", opc)
	fmt.Printf("SQN      = %x\n", sqn)
	fmt.Printf("AMF      = %x\n", amf)
	fmt.Printf("RAND     = %x\n", rand)
	fmt.Println()

	fmt.Printf("-------- MILENAGE ops @ UDM --------\n")
	m := milenage.NewWithOPc(k, opc, rand, sqn, amf)

	maca, err := m.F1()
	if err != nil {
		log.Fatalf("F1() failed: %+v", err)
	}
	fmt.Printf("MAC-A    = %x\n", maca)

	xres, ck, ik, ak, err := m.F2345()
	if err != nil {
		log.Fatalf("F2345() failed: %+v", err)
	}
	fmt.Printf("CK       = %x\n", ck)
	fmt.Printf("IK       = %x\n", ik)
	fmt.Printf("AK       = %x\n", ak)
	fmt.Printf("xRES     = %x\n", xres)

	m.RESStar, err = m.ComputeRESStar((*imsis)[0:3], (*imsis)[3:5])
	xRESStar := m.RESStar
	if err != nil {
		log.Fatalf("Failed to compute RESStar: %+v", err)
	}
	fmt.Printf("xRESStar = %x\n", xRESStar)

	autn, err := m.GenerateAUTN()
	if err != nil {
		log.Fatalf("GenerateAUTN() failed: %+v", err)
	}
	fmt.Printf("AUTN     = %x\n", autn)

	// Resync stuff
	// macs, err := m.F1Star(m.SQN, m.AMF)
	// if err != nil {
	// 	log.Fatalf("F1Star() failed: %+v", err)
	// }
	// fmt.Printf("MAC-S    = %x\n", macs)

	// aks, err := m.F5Star()
	// if err != nil {
	// 	log.Fatalf("F5Star() failed: %+v", err)
	// }
	// fmt.Printf("AKS      = %x\n", aks)

	// auts, err := m.GenerateAUTS()
	// if err != nil {
	// 	log.Fatalf("GenerateAUTS() failed: %+v", err)
	// }
	// fmt.Printf("AUTS     = %x\n", auts)

	snn := fmt.Sprintf("5G:mnc%s.mcc%s.3gppnetwork.org", mnc, mcc)
	a := aka.New(*m, snn, *imsis)

	kausf, err := a.ComputeKAUSF()
	if err != nil {
		log.Fatalf("ComputeKAUSF() failed: %+v", err)
	}
	fmt.Printf("KAUSF    = %x\n", kausf)
	fmt.Println()

	////////////////////////////////////////
	// UDM -> AUSF: RAND, xRESStar, AUTN, KAUSF
	////////////////////////////////////////
	fmt.Printf("******** UDM -> AUSF: RAND, xRESStar, AUTN, KAUSF ********\n")
	fmt.Println()
	fmt.Printf("-------- 5G AKA ops @ AUSF --------\n")

	hxresstar, err := a.ComputeHXRESStar()
	if err != nil {
		log.Fatalf("ComputeHXRESStar() failed: %+v", err)
	}
	fmt.Printf("HXRESStar= %x\n", hxresstar)
	fmt.Println()

	////////////////////////////////////////
	// AUSF -> SEAF: RAND, HXRESStar, AUTN
	////////////////////////////////////////
	fmt.Printf("******** AUSF -> SEAF: RAND, HXRESStar, AUTN ********\n")
	fmt.Println()

	////////////////////////////////////////
	// The serving AMF sends the AKA challenge to the UE
	// The UE sends the AKA response (RESStar) to the AUSF
	// The AUSF verfifies XRESStar matches
	////////////////////////////////////////
	s := `
	The serving AMF sends the AKA challenge to the UE
	The UE sends the AKA response (RESStar) to the serving AMF
	The SEAF verfifies HXRESStar matches
	`
	fmt.Println(s)
	fmt.Println()

	fmt.Printf("-------- 5G AKA ops @ AUSF --------\n")
	kseaf, err := a.ComputeKSEAF()
	if err != nil {
		log.Fatalf("ComputeKSEAF() failed: %+v", err)
	}
	fmt.Printf("KSEAF    = %x\n", kseaf)
	fmt.Println()
	fmt.Printf("******** AUSF -> SEAF: SUPI, KSEAF ********\n")
	fmt.Println()

	fmt.Printf("-------- 5G AKA ops @ SEAF --------\n")
	kamf, err := a.ComputeKAMF()
	if err != nil {
		log.Fatalf("ComputeKAMF() failed: %+v", err)
	}
	fmt.Printf("KAMF     = %x\n", kamf)
}
