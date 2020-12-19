package main

import (
	"math/rand"
	"time"

	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

type PduText struct {
	encoded []byte
	codec   pdutext.Codec
}

func NewEncodedText(encoded []byte, c pdutext.Codec) *PduText {
	return &PduText{
		encoded: encoded,
		codec:   c,
	}
}

// Вытащено из https://github.com/fiorix/go-smpp/blob/6dbf72b9bcea72cea4a035e3a2fc434c4dabaae7/smpp/transmitter.go#L330-L359
// Функция нужна для подготовки многосоставных сообщений smpp провайдером
// https://github.com/yandex/pandora/blob/develop/core/provider/decoder.go#L99-L109
func SplitMessageText(sm *smpp.ShortMessage, text string, enc string) []smpp.ShortMessage {
	codec := codec(enc, text)
	maxLen := 133 // 140-7 (UDH with 2 byte reference number)
	switch codec.(type) {
	case pdutext.GSM7:
		maxLen = 152 // to avoid an escape character being split between payloads
		break
	case pdutext.GSM7Packed:
		maxLen = 132 // to avoid an escape character being split between payloads
		break
	case pdutext.UCS2:
		maxLen = 132 // to avoid a character being split between payloads
		break
	}
	encoded := codec.Encode()
	countParts := int((len(encoded)-1)/maxLen) + 1

	parts := make([]smpp.ShortMessage, 0, countParts)

	if countParts == 1 {
		sm.Text = NewEncodedText(encoded, codec)
		return append(parts, *sm)
	}

	rn := uint16(r.Intn(0xFFFF))

	UDHHeader := make([]byte, 7)
	UDHHeader[0] = 0x06              // length of user data header
	UDHHeader[1] = 0x08              // information element identifier, CSMS 16 bit reference number
	UDHHeader[2] = 0x04              // length of remaining header
	UDHHeader[3] = uint8(rn >> 8)    // most significant byte of the reference number
	UDHHeader[4] = uint8(rn)         // least significant byte of the reference number
	UDHHeader[5] = uint8(countParts) // total number of message parts

	sm.ESMClass = 0x40

	for i := 0; i < countParts; i++ {
		UDHHeader[6] = uint8(i + 1) // current message part
		if i != countParts-1 {
			sm.Text = NewEncodedText(append(UDHHeader, encoded[i*maxLen:(i+1)*maxLen]...), codec)
		} else {
			sm.Text = NewEncodedText(append(UDHHeader, encoded[i*maxLen:]...), codec)
		}
		parts = append(parts, *sm)
	}
	return parts
}

func codec(enc string, raw string) pdutext.Codec {
	switch enc {
	case "ucs2":
		return pdutext.UCS2(raw)
	case "latin1":
		return pdutext.Latin1(raw)
	default:
		return pdutext.Raw(raw)
	}
}

func (s *PduText) Type() pdutext.DataCoding {
	return s.codec.Type()
}

func (s *PduText) Encode() []byte {
	return s.encoded
}

func (s *PduText) Decode() []byte {
	return s.codec.Decode()
}
