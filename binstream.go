package binproto

import (
	"encoding/binary"
	"errors"
	"github.com/silvasur/kagus"
	"io"
	"sync"
)

// BinstreamReader reads a binary stream from a binproto stream.
type BinstreamReader struct {
	r      io.Reader
	err    error
	toread int
	surMu  *sync.Mutex // The mutex of the parent SimpleUnitReader
}

// Read implements io.Reader.
func (bsr *BinstreamReader) Read(p []byte) (int, error) {
	if bsr.err != nil {
		return 0, bsr.err
	}

	if bsr.toread == 0 {
		var _toread int32
		if err := binary.Read(bsr.r, binary.LittleEndian, &_toread); err != nil {
			bsr.err = err
			return 0, err
		}

		if _toread < 0 {
			bsr.toread = -1
			bsr.err = io.EOF
			bsr.surMu.Unlock() // TODO: Unlock on other conditions?
			return 0, io.EOF
		}

		bsr.toread = int(_toread)
	}

	want := len(p)
	if bsr.toread < want {
		want = bsr.toread
	}
	n, err := bsr.r.Read(p[:want])

	switch err {
	case nil:
		bsr.toread -= n
		return n, nil
	case io.EOF:
		// NOTE: Perhaps we should log this? IDK...
		bsr.err = errors.New("binstream terminated abnormally")
		return n, bsr.err
	}

	bsr.err = err
	return n, err
}

// FastForward skips to the end of the stream. Use this, if the data is useless.
func (bsr *BinstreamReader) FastForward() error {
	nirvana := kagus.NewNirvanaWriter()
	_, err := io.Copy(nirvana, bsr)
	return err
}

// BinstreamReader writes a binary stream to a binproto stream.
type BinstreamWriter struct {
	w   io.Writer
	err error
}

// Write implements io.Writer.
func (bsw *BinstreamWriter) Write(p []byte) (int, error) {
	if bsw.err != nil {
		return 0, bsw.err
	}

	l := len(p)
	if l == 0 {
		return 0, nil
	}

	if err := binary.Write(bsw.w, binary.LittleEndian, int32(l)); err != nil {
		bsw.err = err
		return 0, err
	}

	n, err := bsw.w.Write(p)
	if err != nil {
		bsw.err = err
	}

	return n, err
}

// Close implements io.Closer. You MUST close a stream, so it is terminated properly.
func (bsw *BinstreamWriter) Close() error {
	switch bsw.err {
	case nil:
	case io.EOF:
		return nil
	default:
		return bsw.err
	}
	if err := binary.Write(bsw.w, binary.LittleEndian, int32(-1)); err != nil {
		return err
	}

	bsw.err = io.EOF
	return nil
}
