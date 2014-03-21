package binproto

import (
	"bytes"
	"io"
	"testing"
)

func TestDemux(t *testing.T) {
	r := bytes.NewReader([]byte{
		0x02, 0x01, 0x00, // Answer(1)
		0x06,       // List
		0x06,       // List
		0x0d, 0x01, // Byte(1)
		0x0d, 0x02, // Byte(2)
		0x0b,       // Term
		0x06,       // List
		0x0d, 0x03, // Byte(3)
		0x0d, 0x04, // Byte(4)
		0x0b,             // Term
		0x0b,             // Term
		0x03, 0x02, 0x00, // Event(2)
		0x0d, 0x2a, // Byte(42)
		0x02, 0x03, 0x00, // Event(3)
		0x00}) // Nil

	ur := NewSimpleUnitReader(r)
	demux := NewDemux(ur)

	events := demux.Events()
	other := demux.Other()

	_code, err := ReadExpect(other, UTAnswer)
	if err != nil {
		t.Fatalf("Could not read an Answer from other: %s", err)
	}
	if code := _code.(uint16); code != 1 {
		t.Errorf("Expected code 1, got %d.", code)
	}

	if err := SkipNext(other); err != nil {
		t.Fatalf("Error while skipping data: %s", err)
	}

	_code, err = ReadExpect(events, UTEvent)
	if err != nil {
		t.Fatalf("Could not read an event: %s", err)
	}
	if code := _code.(uint16); code != 2 {
		t.Errorf("Expected code 2, got %d.", code)
	}
	_b, err := ReadExpect(events, UTByte)
	if err != nil {
		t.Fatalf("Could not read event data: %s", err)
	}
	if b := _b.(byte); b != 42 {
		t.Errorf("Unexpected event data. Want 42, got: %d", b)
	}

	_code, err = ReadExpect(other, UTAnswer)
	if err != nil {
		t.Fatalf("Could not read an Answer from other (2): %s", err)
	}
	if code := _code.(uint16); code != 3 {
		t.Errorf("Expected code 3, got %d.", code)
	}

	if err := SkipNext(other); err != nil {
		t.Fatalf("Error while skipping data (2): %s", err)
	}

	if _, _, err := other.ReadUnit(); err != io.EOF {
		t.Errorf("Expected io.EOF, got: %s", err)
	}
}
