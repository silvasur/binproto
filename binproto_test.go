package binproto

import (
	"bytes"
	"io/ioutil"
	"testing"
)

var data = []byte{
	0x01, 0x2a, 0x00, // Request(42)
	0x08,       // IdKVMap
	0x09, 0x10, // UKey (16)
	0x04, 0x02, 0x00, 0x00, 0x00, 'h', 'i', // Bin(hi)
	0x09, 0x01, // UKey(1)
	0x0a,                                            // BinStream
	0x05, 0x00, 0x00, 0x00, 'h', 'e', 'l', 'l', 'o', // BinStream chunk (hello)
	0x02, 0x00, 0x00, 0x00, ',', ' ', // BinStream chunk (, )
	0x06, 0x00, 0x00, 0x00, 'w', 'o', 'r', 'l', 'd', '!', // BinStream chunk (world!)
	0xff, 0xff, 0xff, 0xff, // Terminating BinStream
	0x09, 0x02, // UKey(2)
	0x06,                                                 // List
	0x05, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Number(1)
	0x05, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Number(2)
	0x0b,       // Term
	0x09, 0x03, // UKey(3)
	0x07,                                        // TextKVMap
	0x04, 0x03, 0x00, 0x00, 0x00, 'f', 'o', 'o', // Bin(foo)
	0x04, 0x03, 0x00, 0x00, 0x00, 'b', 'a', 'r', // Bin(bar)
	0x0b,       // Term
	0x09, 0x04, // UKey(4)
	0x0c, 0x00, // UBool(false)
	0x09, 0x05, // UKey(5)
	0x0c, 0x01, // UBool(true)
	0x09, 0x06, // UKey(6)
	0x0d, 0x0a, // UByte(10)
	0x0b} // Term

func chkUnitType(t *testing.T, recv, expected UnitType) {
	if recv != expected {
		t.Fatalf("Unit has type %s, %s expected.", recv, expected)
	}
}

func readExpect2(t *testing.T, ur UnitReader, expected UnitType) interface{} {
	data, err := ReadExpect(ur, expected)
	if err != nil {
		t.Fatalf("Error from ReadExpect: %s", err)
	}

	return data
}

func TestReading(t *testing.T) {
	r := bytes.NewReader(data)
	ur := NewSimpleUnitReader(r)

	data := readExpect2(t, ur, UTRequest)
	if code := data.(uint16); code != 42 {
		t.Errorf("Request had code %d, not 42.", code)
	}

	readExpect2(t, ur, UTIdKVMap)

	idkvp, err := ReadIdKVPair(ur)
	if err != nil {
		t.Fatalf("Could not read IdKVPair: %s", err)
	}
	if (idkvp.Key != 16) || (idkvp.ValueType != UTBin) || (!bytes.Equal(idkvp.ValuePayload.([]byte), []byte("hi"))) {
		t.Errorf("Wrong idkvp content: %v", idkvp)
	}

	idkvp, err = ReadIdKVPair(ur)
	if err != nil {
		t.Fatalf("Could not read IdKVPair: %s", err)
	}
	if (idkvp.Key != 1) || (idkvp.ValueType != UTBinStream) {
		t.Fatalf("Wrong key or value type: %d, %s", idkvp.Key, idkvp.ValueType)
	}
	d, err := ioutil.ReadAll(idkvp.ValuePayload.(*BinstreamReader))
	if err != nil {
		t.Fatalf("BinstreamReader failed: %s", err)
	}
	if !bytes.Equal(d, []byte("hello, world!")) {
		t.Errorf("Wrong Binstream data: %v", d)
	}

	idkvp, err = ReadIdKVPair(ur)
	if err != nil {
		t.Fatalf("Could not read IdKVPair: %s", err)
	}
	if (idkvp.Key != 2) || (idkvp.ValueType != UTList) {
		t.Fatalf("Wrong key or value type: %d, %s", idkvp.Key, idkvp.ValueType)
	}

	if item := readExpect2(t, ur, UTNumber); item.(int64) != 1 {
		t.Errorf("Wrong number in list. Want: 1. Got: %d", item.(int64))
	}

	if item := readExpect2(t, ur, UTNumber); item.(int64) != 2 {
		t.Errorf("Wrong number in list. Want: 2. Got: %d", item.(int64))
	}

	readExpect2(t, ur, UTTerm)

	idkvp, err = ReadIdKVPair(ur)
	if err != nil {
		t.Fatalf("Could not read IdKVPair: %s", err)
	}
	if (idkvp.Key != 3) || (idkvp.ValueType != UTTextKVMap) {
		t.Fatalf("Wrong key or value type: %d, %s", idkvp.Key, idkvp.ValueType)
	}

	textkvp, err := ReadTextKVPair(ur)
	if err != nil {
		t.Fatalf("Could not read TextKVPair: %s", err)
	}
	if (textkvp.Key != "foo") || (textkvp.ValueType != UTBin) || (!bytes.Equal(textkvp.ValuePayload.([]byte), []byte("bar"))) {
		t.Errorf("Wrong textkvp content: %v", textkvp)
	}

	if _, err = ReadTextKVPair(ur); err != Terminated {
		t.Fatal("TextKVMap not terminated?")
	}

	idkvp, err = ReadIdKVPair(ur)
	if err != nil {
		t.Fatalf("Could not read IdKVPair: %s", err)
	}
	if (idkvp.Key != 4) || (idkvp.ValueType != UTBool) {
		t.Fatalf("Wrong key or value type: %d, %s", idkvp.Key, idkvp.ValueType)
	}
	if idkvp.ValuePayload.(bool) != false {
		t.Error("Got true, want false")
	}

	idkvp, err = ReadIdKVPair(ur)
	if err != nil {
		t.Fatalf("Could not read IdKVPair: %s", err)
	}
	if (idkvp.Key != 5) || (idkvp.ValueType != UTBool) {
		t.Fatalf("Wrong key or value type: %d, %s", idkvp.Key, idkvp.ValueType)
	}
	if idkvp.ValuePayload.(bool) != true {
		t.Error("Got false, want true")
	}

	idkvp, err = ReadIdKVPair(ur)
	if err != nil {
		t.Fatalf("Could not read IdKVPair: %s", err)
	}
	if (idkvp.Key != 6) || (idkvp.ValueType != UTByte) {
		t.Fatalf("Wrong key or value type: %d, %s", idkvp.Key, idkvp.ValueType)
	}
	if val := idkvp.ValuePayload.(byte); val != 10 {
		t.Errorf("Got byte %d, want 10.", val)
	}

	if _, err = ReadIdKVPair(ur); err != Terminated {
		t.Fatal("IdKVMap not terminated?")
	}
}

func chkerr(t *testing.T, err error, whatfailed string) {
	if err != nil {
		t.Fatalf("%s failed: %s", whatfailed, err)
	}
}

func TestWriting(t *testing.T) {
	w := new(bytes.Buffer)

	chkerr(t, InitRequest(w, 42), "InitRequest")
	chkerr(t, InitIdKVMap(w), "InitIdKVMap")

	chkerr(t, SendUKey(w, 16), "SendUKey")
	chkerr(t, SendBin(w, []byte("hi")), "SendBin")

	chkerr(t, SendUKey(w, 1), "SendUKey")

	bsw, err := InitBinStream(w)
	if err != nil {
		t.Fatalf("Could not init a BinstremWriter: %s", err)
	}
	if _, err := bsw.Write([]byte("hello")); err != nil {
		t.Fatalf("Could not write chunk to bsw: %s", err)
	}
	if _, err := bsw.Write([]byte(", ")); err != nil {
		t.Fatalf("Could not write chunk to bsw: %s", err)
	}
	if _, err := bsw.Write([]byte("world!")); err != nil {
		t.Fatalf("Could not write chunk to bsw: %s", err)
	}
	if err := bsw.Close(); err != nil {
		t.Fatalf("Could not close bsw: %s", err)
	}

	chkerr(t, SendUKey(w, 2), "SendUKey")
	chkerr(t, InitList(w), "InitList")
	chkerr(t, SendNumber(w, 1), "SendNumber")
	chkerr(t, SendNumber(w, 2), "SendNumber")
	chkerr(t, SendTerm(w), "SendTerm")

	chkerr(t, SendUKey(w, 3), "SendUKey")
	chkerr(t, InitTextKVMap(w), "InitTextKVMap")
	chkerr(t, SendTextKey(w, "foo"), "SendTextKey")
	chkerr(t, SendBin(w, []byte("bar")), "SendBin")

	chkerr(t, SendTerm(w), "SendTerm")

	chkerr(t, SendUKey(w, 4), "SendUKey")
	chkerr(t, SendBool(w, false), "SendBool")

	chkerr(t, SendUKey(w, 5), "SendUKey")
	chkerr(t, SendBool(w, true), "SendBool")

	chkerr(t, SendUKey(w, 6), "SendUKey")
	chkerr(t, SendByte(w, 10), "SendByte")

	chkerr(t, SendTerm(w), "SendTerm")

	if !bytes.Equal(w.Bytes(), data) {
		t.Errorf("Wrong data constructed, got: %v", w.Bytes())
	}
}

func TestSkipping(t *testing.T) {
	skipdata := []byte{
		0x08,       // IdKVMap
		0x09, 0x01, // UKey(1)
		0x06,                                                 // List
		0x06,                                                 // List
		0x05, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Number(1)
		0x05, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Number(2)
		0x0b,                                                 // Term
		0x06,                                                 // List
		0x05, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Number(3)
		0x05, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Number(4)
		0x0b,       // Term
		0x0b,       // Term
		0x09, 0x02, // UKey(2)
		0x07,                                        // TextKVMap
		0x04, 0x03, 0x00, 0x00, 0x00, 'f', 'o', 'o', // Bin(foo)
		0x0a,                                            // BinStream
		0x05, 0x00, 0x00, 0x00, 'h', 'e', 'l', 'l', 'o', // BinStream chunk (hello)
		0x02, 0x00, 0x00, 0x00, ',', ' ', // BinStream chunk (, )
		0x06, 0x00, 0x00, 0x00, 'w', 'o', 'r', 'l', 'd', '!', // BinStream chunk (world!)
		0xff, 0xff, 0xff, 0xff, // Terminating BinStream
		0x0b,                                                 // Term
		0x0b,                                                 // Term
		0x05, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // Number(8)

	r := bytes.NewReader(skipdata)
	ur := NewSimpleUnitReader(r)

	if err := SkipNext(ur); err != nil {
		t.Fatalf("Skipping failed: %s", err)
	}

	if ut, data, err := ur.ReadUnit(); (err != nil) || (ut != UTNumber) || (data.(int64) != 8) {
		t.Errorf("ReadUnit returned with unexpected data: %s, %v, %s", ut, data, err)
	}
}
