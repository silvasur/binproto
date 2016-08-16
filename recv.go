package binproto

import (
	"encoding/binary"
	"github.com/silvasur/kagus"
	"io"
	"sync"
)

// UnitReader is an interface with the ReadUnit function, which ist the basic reading function of the binproto.
// ReadUnit reads the next binproto unit from the reader. The second output has a different meaning for each unit type:
//
//     UTRequest, UTAnswer, UTEvent - uint16
//     UTBin                        - []byte
//     UTNumber                     - int64
//     UTUKey, UTByte               - byte
//     UTBinStream                  - *BinStreamReader
//     UTBool                       - bool
//
// A UnitReader implementation should usually wrap the SimpleUnitReader implementation.
type UnitReader interface {
	ReadUnit() (UnitType, interface{}, error)
}

// SimpleUnitReader is a UnitReader implementation that gets its data from an io.Reader.
type SimpleUnitReader struct {
	r  io.Reader
	mu *sync.Mutex
}

func NewSimpleUnitReader(r io.Reader) *SimpleUnitReader {
	return &SimpleUnitReader{
		r:  r,
		mu: new(sync.Mutex)}
}

func (sur *SimpleUnitReader) ReadUnit() (UnitType, interface{}, error) {
	r := sur.r

	sur.mu.Lock()
	doUnlock := true
	defer func() {
		if doUnlock {
			sur.mu.Unlock()
		}
	}()

	_ut, err := kagus.ReadByte(r)
	if err != nil {
		return 0, nil, err
	}

	ut := UnitType(_ut)
	switch ut {
	case UTNil:
		return ut, nil, nil
	case UTRequest, UTAnswer, UTEvent:
		var code uint16
		if err := binary.Read(r, binary.LittleEndian, &code); err != nil {
			return ut, nil, err
		}
		return ut, code, nil
	case UTBin:
		var l uint32
		if err := binary.Read(r, binary.LittleEndian, &l); err != nil {
			return ut, nil, err
		}
		buf := make([]byte, l)
		_, err := io.ReadFull(r, buf)
		if err != nil {
			return ut, nil, err
		}
		return ut, buf, nil
	case UTNumber:
		var n int64
		if err := binary.Read(r, binary.LittleEndian, &n); err != nil {
			return ut, nil, err
		}
		return ut, n, nil
	case UTList, UTTextKVMap, UTIdKVMap:
		return ut, nil, nil
	case UTUKey, UTByte:
		k, err := kagus.ReadByte(r)
		return ut, k, err
	case UTBinStream:
		doUnlock = false
		return ut, &BinstreamReader{r: r, surMu: sur.mu}, nil
	case UTTerm:
		return ut, nil, nil
	case UTBool:
		_b, err := kagus.ReadByte(r)
		b := true
		if _b == 0 {
			b = false
		}
		return ut, b, err
	}

	return ut, nil, UnknownUnit
}

// IdKVPair will be returned from ReadIdKVPair. ValueType and ValuePayload are the first two outputs of ReadUnit for the value.
type IdKVPair struct {
	Key          uint8
	ValueType    UnitType
	ValuePayload interface{}
}

// ReadIdKVPair reads a UKey + any unit pair. err will be Terminated, if this was the last KVPair.
func ReadIdKVPair(ur UnitReader) (kvp IdKVPair, err error) {
	var ut UnitType
	var data interface{}
	ut, data, err = ur.ReadUnit()
	if err != nil {
		return
	}

	switch ut {
	case UTUKey:
		kvp.Key = data.(byte)
	case UTTerm:
		err = Terminated
		return
	default:
		err = UnexpectedUnit
		return
	}

	kvp.ValueType, kvp.ValuePayload, err = ur.ReadUnit()
	return
}

// TextKVPair will be returned from ReadTextKVPair. ValueType and ValuePayload are the first two outputs of ReadUnit for the value.
type TextKVPair struct {
	Key          string
	ValueType    UnitType
	ValuePayload interface{}
}

// ReadTextKVPair reads a Bin(as string) + any unit pair. err will be Terminated, if this was the last KVPair.
func ReadTextKVPair(ur UnitReader) (kvp TextKVPair, err error) {
	var ut UnitType
	var data interface{}
	ut, data, err = ur.ReadUnit()
	if err != nil {
		return
	}

	switch ut {
	case UTBin:
		kvp.Key = string(data.([]byte))
	case UTTerm:
		err = Terminated
		return
	default:
		err = UnexpectedUnit
		return
	}

	kvp.ValueType, kvp.ValuePayload, err = ur.ReadUnit()
	return
}

const maxSkipDepth = 16

func skipUnit(ur UnitReader, ut UnitType, data interface{}, revDepth int) error {
	if revDepth == 0 {
		return TooDeeplyNested
	}

	switch ut {
	case UTNil, UTRequest, UTAnswer, UTEvent, UTBin, UTNumber, UTUKey, UTTerm, UTBool, UTByte:
		return nil
	case UTList:
		for {
			nUt, nData, nErr := ur.ReadUnit()
			if nErr != nil {
				return nErr
			}

			if nUt == UTTerm {
				return nil
			}

			if err := skipUnit(ur, nUt, nData, revDepth-1); err != nil {
				return err
			}
		}
	case UTTextKVMap:
		for {
			switch kvp, err := ReadTextKVPair(ur); err {
			case nil:
				if err := skipUnit(ur, kvp.ValueType, kvp.ValuePayload, revDepth-1); err != nil {
					return err
				}
			case Terminated:
				return nil
			default:
				return err
			}
		}
	case UTIdKVMap:
		for {
			switch kvp, err := ReadIdKVPair(ur); err {
			case nil:
				if err := skipUnit(ur, kvp.ValueType, kvp.ValuePayload, revDepth-1); err != nil {
					return err
				}
			case Terminated:
				return nil
			default:
				return err
			}
		}
	case UTBinStream:
		bsr := data.(*BinstreamReader)
		return bsr.FastForward()
	}

	return UnexpectedUnit
}

// SkipUnit skips the current unit recursively. ut and data are the first two outputs of ReadUnit.
// If the structure is nested too deeply, this function will abort with TooDeeplyNested.
func SkipUnit(ur UnitReader, ut UnitType, data interface{}) error {
	return skipUnit(ur, ut, data, maxSkipDepth)
}

// SkipNext is ReadNext + SkipUnit.
func SkipNext(ur UnitReader) error {
	ut, data, err := ur.ReadUnit()
	if err != nil {
		return err
	}
	return SkipUnit(ur, ut, data)
}

// ReadExpect reads a unit and tests, if the type is te expected type.
// If not, UnexpectedUnit is returned and the read unit will be skipped
// (if this fails, the reason for that is returned instead of UnexpectedError).
func ReadExpect(ur UnitReader, expected UnitType) (interface{}, error) {
	ut, data, err := ur.ReadUnit()
	if err != nil {
		return nil, err
	}

	if ut == expected {
		return data, nil
	}

	if err := SkipUnit(ur, ut, data); err != nil {
		return nil, err
	}

	return nil, UnexpectedUnit
}
