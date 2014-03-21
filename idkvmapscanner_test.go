package binproto

import (
	"bytes"
	"io"
	"testing"
)

func TestRegularIdKVMap(t *testing.T) {
	// We assume that we are already in the map
	testdata := []byte{
		0x09, 0x01, // UKey(1)
		0x05, 0x2a, 0, 0, 0, 0, 0, 0, 0, // Number(42)
		0x09, 0x02, // UKey(2)
		0x0c, 0x00, // Bool(false)
		0x09, 0x03, // UKey(3)
		0x06,                            // List, for testing skipping unknown fields
		0x05, 0x01, 0, 0, 0, 0, 0, 0, 0, // Number(1)
		0x05, 0x02, 0, 0, 0, 0, 0, 0, 0, // Number(2)
		0x05, 0x03, 0, 0, 0, 0, 0, 0, 0, // Number(3)
		0x0b,       // Term
		0x09, 0x04, // UKey(4)
		0x04, 0x02, 0, 0, 0, 'h', 'i', // Bin("hi")
		0x0b} // Term

	r := bytes.NewReader(testdata)
	ur := NewSimpleUnitReader(r)

	var n1 int64
	var b2 bool
	var bs4 []byte

	var seen1, seen2, seen4, seen5 bool

	err := ScanIdKVMap(ur, map[byte]UKeyGetter{
		1: {UTNumber, false, ActionStoreNumber(&n1), &seen1},
		2: {UTBool, false, ActionStoreBool(&b2), &seen2},
		4: {UTBin, false, ActionStoreBin(&bs4), &seen4},
		5: {UTNil, true, ActionSkip(UTNil), &seen5}}, false)

	if err != nil {
		t.Errorf("Did not expect error, got: %s", err)
	}

	if n1 != 42 {
		t.Errorf("n1 wrong, got %d, want 42", n1)
	}

	if b2 != false {
		t.Error("b2 wrong, want false")
	}

	if !bytes.Equal(bs4, []byte("hi")) {
		t.Errorf("bs4 wrong: %v", bs4)
	}

	if !(seen1 && seen2 && seen4 && (!seen5)) {
		t.Errorf("Unexpected values for seen* vars: %v %v %v %v", seen1, seen2, seen4, seen5)
	}

	if b, err := r.ReadByte(); err != io.EOF {
		t.Errorf("Expected EOF for reader r, got %x, %v", b, err)
	}
}

func TestFailOnUnknown(t *testing.T) {
	// We assume that we are already in the map
	testdata := []byte{
		0x09, 0x01, // UKey(1)
		0x05, 0x2a, 0, 0, 0, 0, 0, 0, 0, // Number(42)
		0x09, 0x02, // UKey(2)
		0x0c, 0x00, // Bool(false)
		0x09, 0x03, // UKey(3)
		0x06,                            // List, for testing skipping unknown fields
		0x05, 0x01, 0, 0, 0, 0, 0, 0, 0, // Number(1)
		0x05, 0x02, 0, 0, 0, 0, 0, 0, 0, // Number(2)
		0x05, 0x03, 0, 0, 0, 0, 0, 0, 0, // Number(3)
		0x0b,       // Term
		0x09, 0x04, // UKey(4)
		0x04, 0x02, 0, 0, 0, 'h', 'i', // Bin("hi")
		0x0b} // Term

	r := bytes.NewReader(testdata)
	ur := NewSimpleUnitReader(r)

	err := ScanIdKVMap(ur, map[byte]UKeyGetter{
		1: {UTNumber, false, ActionSkip(UTNumber), nil},
		2: {UTBool, false, ActionSkip(UTBool), nil},
		4: {UTBin, false, ActionSkip(UTBin), nil},
		5: {UTNil, true, ActionSkip(UTNil), nil}}, true)

	if err != UnknownKey {
		t.Errorf("Got wrong error: %s", err)
	}

	if b, err := r.ReadByte(); err != io.EOF {
		t.Errorf("Expected EOF for reader r, got %x, %v", b, err)
	}
}

func TestMissingMandatory(t *testing.T) {
	// We assume that we are already in the map
	testdata := []byte{
		0x09, 0x01, // UKey(1)
		0x05, 0x2a, 0, 0, 0, 0, 0, 0, 0, // Number(42)
		0x09, 0x03, // UKey(3)
		0x06,                            // List, for testing skipping unknown fields
		0x05, 0x01, 0, 0, 0, 0, 0, 0, 0, // Number(1)
		0x05, 0x02, 0, 0, 0, 0, 0, 0, 0, // Number(2)
		0x05, 0x03, 0, 0, 0, 0, 0, 0, 0, // Number(3)
		0x0b,       // Term
		0x09, 0x04, // UKey(4)
		0x04, 0x02, 0, 0, 0, 'h', 'i', // Bin("hi")
		0x0b} // Term

	r := bytes.NewReader(testdata)
	ur := NewSimpleUnitReader(r)

	err := ScanIdKVMap(ur, map[byte]UKeyGetter{
		1: {UTNumber, false, ActionSkip(UTNumber), nil},
		2: {UTBool, false, ActionSkip(UTBool), nil},
		4: {UTBin, false, ActionSkip(UTBin), nil},
		5: {UTNil, true, ActionSkip(UTNil), nil}}, false)

	if err != KeyMissing {
		t.Errorf("Got wrong error: %s", err)
	}

	if b, err := r.ReadByte(); err != io.EOF {
		t.Errorf("Expected EOF for reader r, got %x, %v", b, err)
	}
}
