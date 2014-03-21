package binproto

import (
	"encoding/binary"
	"io"
)

func SendNil(w io.Writer) error {
	_, err := w.Write([]byte{UTNil})
	return err
}

func sendRAE(w io.Writer, what UnitType, code uint16) error {
	if _, err := w.Write([]byte{byte(what)}); err != nil {
		return err
	}

	return binary.Write(w, binary.LittleEndian, code)
}

func InitRequest(w io.Writer, code uint16) error { return sendRAE(w, UTRequest, code) }
func InitAnswer(w io.Writer, code uint16) error  { return sendRAE(w, UTAnswer, code) }
func InitEvent(w io.Writer, code uint16) error   { return sendRAE(w, UTEvent, code) }

func SendBin(w io.Writer, bindata []byte) error {
	if _, err := w.Write([]byte{UTBin}); err != nil {
		return err
	}

	if err := binary.Write(w, binary.LittleEndian, uint32(len(bindata))); err != nil {
		return err
	}

	_, err := w.Write(bindata)
	return err
}

func SendNumber(w io.Writer, n int64) error {
	if _, err := w.Write([]byte{UTNumber}); err != nil {
		return err
	}

	return binary.Write(w, binary.LittleEndian, n)
}

func InitList(w io.Writer) error {
	_, err := w.Write([]byte{UTList})
	return err
}

func InitTextKVMap(w io.Writer) error {
	_, err := w.Write([]byte{UTTextKVMap})
	return err
}

func InitIdKVMap(w io.Writer) error {
	_, err := w.Write([]byte{UTIdKVMap})
	return err
}

func sendTypedByte(w io.Writer, t UnitType, b byte) error {
	_, err := w.Write([]byte{byte(t), b})
	return err
}

func SendUKey(w io.Writer, key byte) error {
	return sendTypedByte(w, UTUKey, key)
}

func SendBool(w io.Writer, b bool) error {
	if b {
		return sendTypedByte(w, UTBool, 1)
	}
	return sendTypedByte(w, UTBool, 0)
}

func SendByte(w io.Writer, b byte) error {
	return sendTypedByte(w, UTByte, b)
}

func SendTextKey(w io.Writer, key string) error {
	return SendBin(w, []byte(key))
}

func SendTerm(w io.Writer) error {
	_, err := w.Write([]byte{UTTerm})
	return err
}

func InitBinStream(w io.Writer) (*BinstreamWriter, error) {
	_, err := w.Write([]byte{UTBinStream})
	if err != nil {
		return nil, err
	}

	return &BinstreamWriter{w: w}, nil
}
