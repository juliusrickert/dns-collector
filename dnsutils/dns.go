package dnsutils

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const DnsLen = 12

var (
	Rdatatypes = map[int]string{
		0:     "NONE",
		1:     "A",
		2:     "NS",
		3:     "MD",
		4:     "MF",
		5:     "CNAME",
		6:     "SOA",
		7:     "MB",
		8:     "MG",
		9:     "MR",
		10:    "NULL",
		11:    "WKS",
		12:    "PTR",
		13:    "HINFO",
		14:    "MINFO",
		15:    "MX",
		16:    "TXT",
		17:    "RP",
		18:    "AFSDB",
		19:    "X25",
		20:    "ISDN",
		21:    "RT",
		22:    "NSAP",
		23:    "NSAP_PTR",
		24:    "SIG",
		25:    "KEY",
		26:    "PX",
		27:    "GPOS",
		28:    "AAAA",
		29:    "LOC",
		30:    "NXT",
		33:    "SRV",
		35:    "NAPTR",
		36:    "KX",
		37:    "CERT",
		38:    "A6",
		39:    "DNAME",
		41:    "OPT",
		42:    "APL",
		43:    "DS",
		44:    "SSHFP",
		45:    "IPSECKEY",
		46:    "RRSIG",
		47:    "NSEC",
		48:    "DNSKEY",
		49:    "DHCID",
		50:    "NSEC3",
		51:    "NSEC3PARAM",
		52:    "TSLA",
		53:    "SMIMEA",
		55:    "HIP",
		56:    "NINFO",
		59:    "CDS",
		60:    "CDNSKEY",
		61:    "OPENPGPKEY",
		62:    "CSYNC",
		64:    "SVCB",
		65:    "HTTPS",
		99:    "SPF",
		103:   "UNSPEC",
		108:   "EUI48",
		109:   "EUI64",
		249:   "TKEY",
		250:   "TSIG",
		251:   "IXFR",
		252:   "AXFR",
		253:   "MAILB",
		254:   "MAILA",
		255:   "ANY",
		256:   "URI",
		257:   "CAA",
		258:   "AVC",
		259:   "AMTRELAY",
		32768: "TA",
		32769: "DLV",
	}
	Rcodes = map[int]string{
		0:  "NOERROR",
		1:  "FORMERR",
		2:  "SERVFAIL",
		3:  "NXDOMAIN",
		4:  "NOIMP",
		5:  "REFUSED",
		6:  "YXDOMAIN",
		7:  "YXRRSET",
		8:  "NXRRSET",
		9:  "NOTAUTH",
		10: "NOTZONE",
		11: "DSOTYPENI",
		16: "BADSIG",
		17: "BADKEY",
		18: "BADTIME",
		19: "BADMODE",
		20: "BADNAME",
		21: "BADALG",
		22: "BADTRUNC",
		23: "BADCOOKIE",
	}
)

var ErrDecodeDnsHeaderTooShort = errors.New("malformed pkt, dns payload too short to decode header")
var ErrDecodeDnsLabelInvalidOffset = errors.New("malformed pkt, invalid offset to decode label")
var ErrDecodeDnsLabelInvalidOffsetInfiniteLoop = errors.New("malformed pkt, invalid offset to decode label, infinite loop")
var ErrDecodeDnsLabelTooShort = errors.New("malformed pkt, dns payload too short to get label")
var ErrDecodeQuestionQtypeTooShort = errors.New("malformed pkt, not enough data to decode qtype")
var ErrDecodeDnsAnswerTooShort = errors.New("malformed pkt, not enough data to decode answer")
var ErrDecodeDnsAnswerRdataTooShort = errors.New("malformed pkt, not enough data to decode rdata answer")

func RdatatypeToString(rrtype int) string {
	if value, ok := Rdatatypes[rrtype]; ok {
		return value
	}
	return "UNKNOWN"
}

func RcodeToString(rcode int) string {
	if value, ok := Rcodes[rcode]; ok {
		return value
	}
	return "UNKNOWN"
}

type DnsHeader struct {
	Id      int
	Qr      int
	Opcode  int
	Aa      int
	Tc      int
	Rd      int
	Ra      int
	Z       int
	Ad      int
	Cd      int
	Rcode   int
	Qdcount int
	Ancount int
	Nscount int
	Arcount int
}

/*
	DNS HEADER
									1  1  1  1  1  1
	  0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                      ID                       |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|QR|   Opcode  |AA|TC|RD|RA| Z|AD|CD|   RCODE   |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                    QDCOUNT                    |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                    ANCOUNT                    |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                    NSCOUNT                    |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                    ARCOUNT                    |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
*/

func DecodeDns(payload []byte) (DnsHeader, error) {
	dh := DnsHeader{}

	// before to start, check to be sure to have enough data to decode
	if len(payload) < DnsLen {
		return dh, ErrDecodeDnsHeaderTooShort
	}
	// decode ID
	dh.Id = int(binary.BigEndian.Uint16(payload[:2]))

	// decode flags
	dh.Qr = int(binary.BigEndian.Uint16(payload[2:4]) >> 0xF)
	dh.Opcode = int((binary.BigEndian.Uint16(payload[2:4]) >> (3 + 0x8)) & 0xF)
	dh.Aa = int((binary.BigEndian.Uint16(payload[2:4]) >> (2 + 0x8)) & 1)
	dh.Tc = int((binary.BigEndian.Uint16(payload[2:4]) >> (1 + 0x8)) & 1)
	dh.Rd = int((binary.BigEndian.Uint16(payload[2:4]) >> (0x8)) & 1)
	dh.Cd = int((binary.BigEndian.Uint16(payload[2:4]) >> 4) & 1)
	dh.Ad = int((binary.BigEndian.Uint16(payload[2:4]) >> 5) & 1)
	dh.Z = int((binary.BigEndian.Uint16(payload[2:4]) >> 6) & 1)
	dh.Ra = int((binary.BigEndian.Uint16(payload[2:4]) >> 7) & 1)
	dh.Rcode = int(binary.BigEndian.Uint16(payload[2:4]) & 0xF)

	// decode counters
	dh.Qdcount = int(binary.BigEndian.Uint16(payload[4:6]))
	dh.Ancount = int(binary.BigEndian.Uint16(payload[6:8]))
	dh.Nscount = int(binary.BigEndian.Uint16(payload[8:10]))
	dh.Arcount = int(binary.BigEndian.Uint16(payload[10:12]))

	return dh, nil
}

/*
	DNS QUESTION
								   1  1  1  1  1  1
	 0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                                               |
	/                     QNAME                     /
	/                                               /
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                     QTYPE                     |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                     QCLASS                    |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
*/
func DecodeQuestion(payload []byte) (string, int, int, error) {
	// Decode QNAME
	qname, offset, err := ParseLabels(DnsLen, payload)
	if err != nil {
		return "", 0, 0, err
	}

	// decode QTYPE and support invalid packet, some abuser sends it...
	var qtype uint16
	if len(payload[offset:]) < 4 {
		return "", 0, 0, ErrDecodeQuestionQtypeTooShort
	} else {
		qtype = binary.BigEndian.Uint16(payload[offset : offset+2])
		offset += 4
	}
	return qname, int(qtype), offset, err
}

/*
    DNS ANSWER
	                               1  1  1  1  1  1
	 0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                                               |
	/                                               /
	/                      NAME                     /
	|                                               |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                      TYPE                     |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                     CLASS                     |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                      TTL                      |
	|                                               |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	|                   RDLENGTH                    |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--|
	/                     RDATA                     /
	/                                               /
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+

	PTR can be used on NAME for compression
									1  1  1  1  1  1
	  0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	| 1  1|                OFFSET                   |
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
*/

func DecodeAnswer(ancount int, start_offset int, payload []byte) ([]DnsAnswer, int, error) {
	offset := start_offset
	answers := []DnsAnswer{}

	for i := 0; i < ancount; i++ {
		// Decode NAME
		name, offset_next, err := ParseLabels(offset, payload)
		if err != nil {
			return answers, offset, err
		}

		// before to continue, check we have enough data
		if len(payload[offset_next:]) < 10 {
			return answers, offset, ErrDecodeDnsAnswerTooShort
		}
		// decode TYPE
		t := binary.BigEndian.Uint16(payload[offset_next : offset_next+2])
		// decode CLASS
		class := binary.BigEndian.Uint16(payload[offset_next+2 : offset_next+4])
		// decode TTL
		ttl := binary.BigEndian.Uint32(payload[offset_next+4 : offset_next+8])
		// decode RDLENGTH
		rdlength := binary.BigEndian.Uint16(payload[offset_next+8 : offset_next+10])

		// decode RDATA
		// but before to continue, check we have enough data to decode the rdata
		if len(payload[offset_next+10:]) < int(rdlength) {
			return answers, offset, ErrDecodeDnsAnswerRdataTooShort
		}
		rdata := payload[offset_next+10 : offset_next+10+int(rdlength)]

		// ignore OPT, this type is decoded in the EDNS extension
		if t == 41 {
			continue
		}
		// parse rdata
		rdatatype := RdatatypeToString(int(t))
		parsed, err := ParseRdata(rdatatype, rdata, payload, offset_next+10)
		if err != nil {
			return answers, offset, err
		}

		// finnally append answer to the list
		a := DnsAnswer{
			Name:      name,
			Rdatatype: rdatatype,
			Class:     int(class),
			Ttl:       int(ttl),
			Rdata:     parsed,
		}
		answers = append(answers, a)

		// compute the next offset
		offset = offset_next + 10 + int(rdlength)
	}
	return answers, offset, nil
}

func ParseLabels(offset int, payload []byte) (string, int, error) {
	ptrs := make(map[uint16]int)
	return _ParseLabels(offset, payload, ptrs)
}

func _ParseLabels(offset int, payload []byte, pointers map[uint16]int) (string, int, error) {
	labels := []string{}
	for {
		if offset >= len(payload) {
			return "", 0, ErrDecodeDnsLabelInvalidOffset
		}

		length := int(payload[offset])
		if length == 0 {
			offset++
			break
		}
		// label pointer support ?
		if length>>6 == 3 {
			ptr := binary.BigEndian.Uint16(payload[offset:offset+2]) & 16383
			_, exist := pointers[ptr]
			if exist {
				return "", 0, ErrDecodeDnsLabelInvalidOffsetInfiniteLoop
			} else {
				pointers[ptr] = 1
			}
			label, _, err := _ParseLabels(int(ptr), payload, pointers)
			if err != nil {
				return "", 0, err
			}
			labels = append(labels, label)
			offset += 2
			break

		} else {
			if offset+length+1 >= len(payload) {
				return "", 0, ErrDecodeDnsLabelTooShort
			}
			label := payload[offset+1 : offset+length+1]
			labels = append(labels, string(label))

			offset += length + 1
		}
	}
	return strings.Join(labels[:], "."), offset, nil
}

func ParseRdata(rdatatype string, rdata []byte, payload []byte, rdata_offset int) (string, error) {
	var ret string
	var err error
	switch rdatatype {
	case "A":
		ret, err = ParseA(rdata)
	case "AAAA":
		ret, err = ParseAAAA(rdata)
	case "CNAME":
		ret, err = ParseCNAME(rdata_offset, payload)
	case "MX":
		ret, err = ParseMX(rdata_offset, payload)
	case "SRV":
		ret, err = ParseSRV(rdata_offset, payload)
	case "NS":
		ret, err = ParseNS(rdata_offset, payload)
	case "TXT":
		ret, err = ParseTXT(rdata)
	case "PTR":
		ret, err = ParsePTR(rdata_offset, payload)
	case "SOA":
		ret, err = ParseSOA(rdata_offset, payload)
	default:
		ret = "-"
		err = nil
	}
	return ret, err
}

/*
SOA
								1  1  1  1  1  1
  0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
/                     MNAME                     /
/                                               /
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
/                     RNAME                     /
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    SERIAL                     |
|                                               |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    REFRESH                    |
|                                               |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                     RETRY                     |
|                                               |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    EXPIRE                     |
|                                               |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    MINIMUM                    |
|                                               |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
*/
func ParseSOA(rdata_offset int, payload []byte) (string, error) {
	var offset int

	primaryNS, offset, err := ParseLabels(rdata_offset, payload)
	if err != nil {
		return "", err
	}

	respMailbox, offset, err := ParseLabels(offset, payload)
	if err != nil {
		return "", err
	}
	rdata := payload[offset:]

	serial := binary.BigEndian.Uint32(rdata[0:4])
	refresh := int32(binary.BigEndian.Uint32(rdata[4:8]))
	retry := int32(binary.BigEndian.Uint32(rdata[8:12]))
	expire := int32(binary.BigEndian.Uint32(rdata[12:16]))
	minimum := binary.BigEndian.Uint32(rdata[16:20])

	soa := fmt.Sprintf("%s %s %d %d %d %d %d", primaryNS, respMailbox, serial, refresh, retry, expire, minimum)
	return soa, nil
}

/*
IPv4
								1  1  1  1  1  1
  0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    ADDRESS                    |
|                                               |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
*/
func ParseA(r []byte) (string, error) {
	var ip []string
	for i := 0; i < len(r); i++ {
		ip = append(ip, strconv.Itoa(int(r[i])))
	}
	a := strings.Join(ip, ".")
	return a, nil
}

/*
IPv6
								1  1  1  1  1  1
  0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                                               |
|                                               |
|                                               |
|                    ADDRESS                    |
|                                               |
|                                               |
|                                               |
|                                               |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
*/
func ParseAAAA(rdata []byte) (string, error) {
	var ip []string
	for i := 0; i < len(rdata); i += 2 {
		ip = append(ip, fmt.Sprintf("%x", binary.BigEndian.Uint16(rdata[i:i+2])))
	}
	aaaa := strings.Join(ip, ":")
	return aaaa, nil
}

/*
CNAME
								1  1  1  1  1  1
  0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
/                     NAME                      /
/                                               /
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
*/
func ParseCNAME(rdata_offset int, payload []byte) (string, error) {
	cname, _, err := ParseLabels(rdata_offset, payload)
	if err != nil {
		return "", err
	}
	return cname, err
}

/*
MX
								1  1  1  1  1  1
  0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                  PREFERENCE                   |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
/                   EXCHANGE                    /
/                                               /
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
*/
func ParseMX(rdata_offset int, payload []byte) (string, error) {
	pref := binary.BigEndian.Uint16(payload[rdata_offset : rdata_offset+2])
	host, _, err := ParseLabels(rdata_offset+2, payload)
	if err != nil {
		return "", err
	}
	mx := fmt.Sprintf("%d %s", pref, host)
	return mx, err
}

/*
SRV
								1  1  1  1  1  1
  0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                   PRIORITY                    |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    WEIGHT                     |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                     PORT                      |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|                    TARGET                     |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
*/
func ParseSRV(rdata_offset int, payload []byte) (string, error) {
	priority := binary.BigEndian.Uint16(payload[rdata_offset : rdata_offset+2])
	weight := binary.BigEndian.Uint16(payload[rdata_offset+2 : rdata_offset+4])
	port := binary.BigEndian.Uint16(payload[rdata_offset+4 : rdata_offset+6])
	target, _, err := ParseLabels(rdata_offset+6, payload)
	if err != nil {
		return "", err
	}
	srv := fmt.Sprintf("%d %d %d %s", priority, weight, port, target)
	return srv, err
}

/*
NS
								1  1  1  1  1  1
  0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
/                   NSDNAME                     /
/                                               /
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
*/
func ParseNS(rdata_offset int, payload []byte) (string, error) {
	ns, _, err := ParseLabels(rdata_offset, payload)
	if err != nil {
		return "", err
	}
	return ns, err
}

/*
TXT
								1  1  1  1  1  1
  0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
+--+--+--+--+--+--+--+--+
|         LENGTH        |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
/                   TXT-DATA                    /
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
*/
func ParseTXT(rdata []byte) (string, error) {
	length := int(rdata[0])
	txt := string(rdata[1 : length+1])
	return txt, nil
}

/*
PTR
									1  1  1  1  1  1
		0  1  2  3  4  5  6  7  8  9  0  1  2  3  4  5
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
	/                   PTRDNAME                    /
	+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
*/
func ParsePTR(rdata_offset int, payload []byte) (string, error) {
	ptr, _, err := ParseLabels(rdata_offset, payload)
	if err != nil {
		return "", err
	}
	return ptr, err
}
