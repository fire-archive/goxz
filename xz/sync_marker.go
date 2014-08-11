// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

// Declaration of the magic byte sequence found in XZ files.
var _SYNC_HEADER = []byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00}

type syncMarker struct {
	offset int64
	state  int
}

func newSyncMarker() *syncMarker {
	return new(syncMarker)
}

// Push a buffer of data into the syncMarker searcher which can only identify
// sync marks after the last byte in a synchronization byte sequence.
// This function returns once it finds the first sync sequence in the buffer.
// It always returns the total number of bytes ever consumed by the syncMarker,
// along with the index of the first byte following the sync sequence.
// If no sync sequence was found, the the index returned will be -1.
func (sm *syncMarker) push(buf []byte) (absPos int64, relPos int) {
	for idx, val := range buf {
		if val == _SYNC_HEADER[sm.state] {
			sm.state++
			if sm.state == len(_SYNC_HEADER) {
				sm.state = 0
				sm.offset += int64(idx + 1)
				return sm.offset, idx + 1
			}
		} else {
			sm.state = 0
		}
	}
	sm.offset += int64(len(buf))
	return sm.offset, -1 // Never found a sync-marker
}

// Byte length of the synchronization marker.
func (sm *syncMarker) syncLen() int {
	return len(_SYNC_HEADER)
}

// Byte sequence of the synchronization marker.
func (sm *syncMarker) syncMark() []byte {
	return append([]byte(nil), _SYNC_HEADER...)
}
