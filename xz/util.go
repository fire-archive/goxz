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

func errConvert(err error, errNew error, errChks ...error) error {
	for _, errChk := range errChks {
		if err == errChk {
			return errNew
		}
	}
	return err
}

func errMatch(err error, errChks ...error) bool {
	return err != nil && errConvert(err, nil, errChks...) == nil
}

func errIgnore(err error, errChks ...error) error {
	return errConvert(err, nil, errChks...)
}
