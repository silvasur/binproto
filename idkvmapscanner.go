package binproto

import (
	"errors"
	"fmt"
	"io"
)

// UKeyGetter defines what to do on a UKey. Used by ScanIdKVMap.
type UKeyGetter struct {
	Type     UnitType     // Which type must the unit have?
	Optional bool         // Is this key optional?
	Action   GetterAction // Performing this action
	Captured *bool        // Pointer to variable that should be set to true, if key was found. Can be nil, if this information is not needed.
}

// Errors of ScanIdKVMap.
var (
	KeyMissing           = errors.New("Mandatory key is missing")
	UnknownKey           = errors.New("Unknown key found")
	UnexpectedTypeForKey = errors.New("Unexpected unit type for key")
)

// GetterAction specifies how an Action in UKeyGetter has to look like. Any error will be passed to the caller.
//
// If fatal is true, ScanIdKVMap will not try to cleanly skip the IdKVMap.
// If fatal is false, the action MUST handle all Units in a clear manner, so ScanIdKVMap can continue operating on a valid stream.
//
// fatal will only be checked, if err != nil.
//
// Actions for keys that are present will be executed asap, so you can e.g. NOT rely that all non-optional keys were captured.
type GetterAction func(interface{}, UnitReader) (err error, fatal bool)

// ScanIdKVMap scans a IdKVMap for keys, tests some properties of the values and will trigger a user given action.
// It also takes care of optional/mandatory keys and will generally make reading the often used IdKVMap less painful.
//
// The input stream must be positioned after the opening UTIdKVMap.
//
// If a key is missing KeyMissing is returned.
// If a key had a value with the wrong type, UnexpectedTypeForKey is returned.
// If a key is unknown AND failOnUnknown is true, UnknownKey is returned.
//
// Other errors either indicate an error in the stream OR an action returned that error.
// Since actions should only return errors when something is really wrong, you should not process the stream any further.
func ScanIdKVMap(ur UnitReader, getters map[byte]UKeyGetter, failOnUnknown bool) (outerr error) {
	seen := make(map[byte]bool)

	skipAll := false
	for {
		ut, data, err := ur.ReadUnit()
		if err != nil {
			return err
		}

		if ut == UTTerm {
			break
		}

		if skipAll {
			if err := SkipUnit(ur, ut, data); err != nil {
				return fmt.Errorf("Error while skipping: %s. Previous outerr was: %s", err, outerr)
			}
			continue
		}

		if ut != UTUKey {
			return fmt.Errorf("Found Unit of type %s (%d) in IdKVMap, expected UTUKey.", ut, ut)
		}

		key := data.(byte)

		ut, data, err = ur.ReadUnit()
		if err != nil {
			return err
		}

		getter, ok := getters[key]
		if !ok {
			if failOnUnknown {
				outerr = UnknownKey
				if err := SkipUnit(ur, ut, data); err != nil {
					return err
				}
				skipAll = true
			} else {
				if err := SkipUnit(ur, ut, data); err != nil {
					return err
				}
			}
			continue
		}

		if ut != getter.Type {
			if err := SkipUnit(ur, ut, data); err != nil {
				return err
			}
			outerr = UnexpectedTypeForKey
			skipAll = true
			continue
		}

		err, fatal := getter.Action(data, ur)
		if err != nil {
			if fatal {
				return err
			}

			outerr = err
			skipAll = true
			continue
		}

		seen[key] = true
		if getter.Captured != nil {
			*(getter.Captured) = true
		}
	}

	if skipAll {
		return
	}

	for key, getter := range getters {
		if !getter.Optional {
			if !seen[key] {
				return KeyMissing
			}
		}
	}

	return nil
}

// ActionSkip builds an action to skip this unit (useful, if you only want to know that a key exists, or for debugging).
func ActionSkip(ut UnitType) GetterAction {
	return func(data interface{}, ur UnitReader) (error, bool) {
		return SkipUnit(ur, ut, data), true // Since second return value will only be inspected, if first is != nil, this is okay.
	}
}

// ActionStoreNumber builds an action for storing a number.
func ActionStoreNumber(n *int64) GetterAction {
	return func(data interface{}, ur UnitReader) (error, bool) {
		*n = data.(int64)
		return nil, false
	}
}

// ActionStoreBin builds an action for storing binary data.
func ActionStoreBin(b *[]byte) GetterAction {
	return func(data interface{}, ur UnitReader) (error, bool) {
		*b = data.([]byte)
		return nil, false
	}
}

// ActionStoreBool builds an action for storing a boolean value.
func ActionStoreBool(b *bool) GetterAction {
	return func(data interface{}, ur UnitReader) (error, bool) {
		*b = data.(bool)
		return nil, false
	}
}

// ActionStoreByte builds an action for storing a byte.
func ActionStoreByte(b *byte) GetterAction {
	return func(data interface{}, ur UnitReader) (error, bool) {
		*b = data.(byte)
		return nil, false
	}
}

// ActionCopyBinStream builds an action that will copy the content of a BinStream to a writer.
func ActionCopyBinStream(w io.Writer) GetterAction {
	return func(data interface{}, ur UnitReader) (error, bool) {
		bsr := data.(*BinstreamReader)
		_, err := io.Copy(w, bsr)
		if err != nil {
			// Try to skip the rest of the binstream in case the error resulted from w.
			if err := bsr.FastForward(); err != nil {
				return err, true // This is a fatal error, since the stream is now corrupted.
			}
			return err, false
		}
		return nil, false
	}
}
