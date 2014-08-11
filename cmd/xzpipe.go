// Copyright 2014, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// A trivial example of pipe-only xz encoder/decoder.
//
// Usage example:
// 	cat test.tar | ./xzpipe > test.tar.xz
// 	cat test.tar.xz | ./xzpipe -d > test.tar
package main

import "os"
import "io"
import "flag"
import "bitbucket.org/rawr/goxz/xz"

func errPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	// Switch compression mode.
	var dec bool
	fs := flag.NewFlagSet("xzpipe", flag.ExitOnError)
	fs.BoolVar(&dec, "d", false, "Decompression mode")
	fs.Parse(os.Args[1:])

	if !dec {
		// Compression mode.
		xz, err := xz.NewWriter(os.Stdout)
		errPanic(err)
		defer xz.Close()

		// Copy all stdin to the compressor.
		_, err = io.Copy(xz, os.Stdin)
		errPanic(err)
	} else {
		// Decompression mode.
		xz, err := xz.NewReader(os.Stdin)
		errPanic(err)
		defer xz.Close()

		// Copy all decompressor output to stdout.
		_, err = io.Copy(os.Stdout, xz)
		errPanic(err)
	}
}
