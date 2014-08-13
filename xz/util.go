// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package xz

func errPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func errRecover(err *error) {
	if ex := recover(); ex != nil {
		if _err, ok := ex.(error); ok {
			(*err) = _err
		} else {
			panic(ex)
		}
	}
}

func errIgnore(err error, errIgns ...error) error {
	for _, errIgn := range errIgns {
		if err == errIgn {
			return nil
		}
	}
	return err
}
