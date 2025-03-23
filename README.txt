main.go output

IMSI     = 001 01 0123456789
K        = 00112233445566778899aabbccddeeff
OPc      = 62e75b8d6fa5bf46ec87a9276f9df54d
SQN      = 1
AMF      = 8000
RAND     = 00112233445566778899aabbccddeeff

-------- MILENAGE ops @ UDM --------
MAC-A    = 4af30b82a8531115
CK       = b379874b3d183d2a21291d439e7761e1
IK       = f4706f66629cf7ddf881d80025bf1255
AK       = de656c8b0bce
xRES     = 700eb2300b2c4799
xRESStar = 31b6d938a5290ccc65bc829f9820a8d9
AUTN     = de656c8b0bcf80004af30b82a8531115
KAUSF    = fe8d2546b6971c510329cd8ae34c177d6569486aa9b71159cc3b5c752a93bd10

******** UDM -> AUSF: RAND, xRESStar, AUTN, KAUSF ********

-------- 5G AKA ops @ AUSF --------
HXRESStar= 3308fb7cf06a35f1cd086b904ce82ecf

******** AUSF -> SEAF: RAND, HXRESStar, AUTN ********


	The serving AMF sends the AKA challenge to the UE
	The UE sends the AKA response (RESStar) to the serving AMF
	The SEAF verfifies HXRESStar matches
	

-------- 5G AKA ops @ AUSF --------
KSEAF    = 442ac77e2366d8084cb447883b03311065ea6bbd8753cf87e92c0669019cf829

******** AUSF -> SEAF: SUPI, KSEAF ********

-------- 5G AKA ops @ SEAF --------
KAMF     = e0c07aacbba7d77ad55efa309882963a9d46dbc9f0045026df89a5d9a30d9915
