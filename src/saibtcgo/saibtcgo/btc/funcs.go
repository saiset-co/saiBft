package btc

import (
	"io"
	"errors"
	"encoding/base64"
	"encoding/binary"
)

func ReadAll(rd io.Reader, b []byte) (er error) {
	var n int
	for i:=0; i<len(b); i+=n {
		n, er = rd.Read(b[i:])
		if er!=nil {
			return
		}
	}
	return
}

// Writes var_length field into the given writer
func WriteVlen(b io.Writer, var_len uint64) {
	if var_len < 0xfd {
		b.Write([]byte{byte(var_len)})
		return
	}
	if var_len < 0x10000 {
		b.Write([]byte{0xfd})
		binary.Write(b, binary.LittleEndian, uint16(var_len))
		return
	}
	if var_len < 0x100000000 {
		b.Write([]byte{0xfe})
		binary.Write(b, binary.LittleEndian, uint32(var_len))
		return
	}
	b.Write([]byte{0xff})
	binary.Write(b, binary.LittleEndian, var_len)
}

// Takes a base64 encoded bitcoin generated signature and decodes it
func ParseMessageSignature(encsig string) (nv byte, sig *Signature, er error) {
	var sd []byte

	sd, er = base64.StdEncoding.DecodeString(encsig)
	if er != nil {
		return
	}

	if len(sd)!=65 {
		er = errors.New("The decoded signature is not 65 bytes long")
		return
	}

	nv = sd[0]

	sig = new(Signature)
	sig.R.SetBytes(sd[1:33])
	sig.S.SetBytes(sd[33:65])

	if nv<27 || nv>34 {
		er = errors.New("nv out of range")
	}

	return
}
