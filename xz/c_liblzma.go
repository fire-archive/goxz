package xz

/*
#cgo LDFLAGS: -llzma
#include <lzma.h>
*/
import "C"

import _ "unsafe"

// Provide thin wrappers around functions and constants provided by the liblzma
// library. The liblzma library by Lasse Collin is used because there are no
// available pure Go implementations that support the newer xz file format.

// Status code constants.
const lzma_OK = C.LZMA_OK
const lzma_STREAM_END = C.LZMA_STREAM_END
const lzma_NO_CHECK = C.LZMA_NO_CHECK
const lzma_UNSUPPORTED_CHECK = C.LZMA_UNSUPPORTED_CHECK
const lzma_GET_CHECK = C.LZMA_GET_CHECK
const lzma_MEM_ERROR = C.LZMA_MEM_ERROR
const lzma_MEMLIMIT_ERROR = C.LZMA_MEMLIMIT_ERROR
const lzma_FORMAT_ERROR = C.LZMA_FORMAT_ERROR
const lzma_OPTIONS_ERROR = C.LZMA_OPTIONS_ERROR
const lzma_DATA_ERROR = C.LZMA_DATA_ERROR
const lzma_BUF_ERROR = C.LZMA_BUF_ERROR
const lzma_PROG_ERROR = C.LZMA_PROG_ERROR

// Flushing mode constants.
const lzma_RUN = C.LZMA_RUN
const lzma_SYNC_FLUSH = C.LZMA_SYNC_FLUSH
const lzma_FULL_FLUSH = C.LZMA_FULL_FLUSH
const lzma_FINISH = C.LZMA_FINISH

// Go error wrapper for underlying liblzma errors.
type zerr struct {
	code int
}

func newZerr(z *lzmaStream, code C.int) error {
	if code == lzma_OK {
		return nil
	} else {
		return &zerr{int(code)}
	}
}

func (ze *zerr) Error() string {
	name := "[UNKNOWN] Unknown error"
	switch ze.code {
	case lzma_STREAM_END:
		name = "[STREAM_END] End of stream was reached"
	case lzma_NO_CHECK:
		name = "[NO_CHECK] Input stream has no integrity check"
	case lzma_UNSUPPORTED_CHECK:
		name = "[UNSUPPORTED_CHECK] Cannot calculate the integrity check"
	case lzma_GET_CHECK:
		name = "[GET_CHECK] Integrity check type is now available"
	case lzma_MEM_ERROR:
		name = "[MEM_ERROR] Cannot allocate memory"
	case lzma_MEMLIMIT_ERROR:
		name = "[MEMLIMIT_ERROR] Memory usage limit was reached"
	case lzma_FORMAT_ERROR:
		name = "[FORMAT_ERROR] File format not recognized"
	case lzma_OPTIONS_ERROR:
		name = "[OPTIONS_ERROR] Invalid or unsupported options"
	case lzma_DATA_ERROR:
		name = "[DATA_ERROR] Data is corrupt"
	case lzma_BUF_ERROR:
		name = "[BUF_ERROR] No progress is possible"
	case lzma_PROG_ERROR:
		name = "[PROG_ERROR] Programming error"
	}
	return name
}

func zerrMatch(err error, errnoMatches ...int) bool {
	if ze, ok := err.(*zerr); ok {
		for _, errMatch := range errnoMatches {
			if ze.code == errMatch {
				return true
			}
		}
	}
	return false
}

func zerrIgnore(err error, errnoIgns ...int) error {
	if zerrMatch(err, errnoIgns...) {
		return nil
	}
	return err
}

// Go wrapper around the liblzma stream struct.
type lzmaStream C.lzma_stream
