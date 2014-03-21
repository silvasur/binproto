// Package binproto provides functions to handle a simple binary protocol.
package binproto

import (
	"errors"
)

type UnitType byte

// Possible UnitType values
const (
	UTNil = iota
	UTRequest
	UTAnswer
	UTEvent
	UTBin
	UTNumber
	UTList
	UTTextKVMap
	UTIdKVMap
	UTUKey
	UTBinStream
	UTTerm
	UTBool
	UTByte
)

func (ut UnitType) String() string {
	switch ut {
	case UTNil:
		return "UTNil"
	case UTRequest:
		return "UTRequest"
	case UTAnswer:
		return "UTAnswer"
	case UTEvent:
		return "UTEvent"
	case UTBin:
		return "UTBin"
	case UTNumber:
		return "UTNumber"
	case UTList:
		return "UTList"
	case UTTextKVMap:
		return "UTTextKVMap"
	case UTIdKVMap:
		return "UTIdKVMap"
	case UTUKey:
		return "UTUKey"
	case UTBinStream:
		return "UTBinStream"
	case UTTerm:
		return "UTTerm"
	case UTBool:
		return "UTBool"
	case UTByte:
		return "UTByte"
	}
	return "Unknown unit"
}

// Errors
var (
	UnknownUnit     = errors.New("Unknown unit received")
	UnexpectedUnit  = errors.New("Unexpected unit received")
	Terminated      = errors.New("List or KVMap terminated")
	TooDeeplyNested = errors.New("Received data is too deeply nested to skip")
)
