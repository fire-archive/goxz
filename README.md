# Parallelized XZ Compression Library for Go #

## Introduction ##

*INCOMPLETE PROJECT*

## Packages ##

* [cmd](http://godoc.org/bitbucket.org/rawr/goxz/cmd): Example executables that
use this library.
* [xz](http://godoc.org/bitbucket.org/rawr/goxz/xz): Library for reading and 
writing the newer XZ file format.
* [lzma](http://godoc.org/bitbucket.org/rawr/goxz/lzma): Library for reading and
writing the obsolete LZMA_ALONE file format.
* [lib](http://godoc.org/bitbucket.org/rawr/goxz/lib): Thin wrappers around the
liblzma C library for use by the xz and lzma packages.

## Theory ##

*To be continued*

## Results ##

*To be continued*

## Frequently asked questions ##

### What exactly is lzma? ###
Depending on the context, the term "lzma" can refer to a number of things.
As a compression _algorithm_, the term refers to the Lempel–Ziv–Markov chain
algorithm that provides lossless data compression. Currently, the lzma algorithm
is implemented as the lzma1 and lzma2 filters.

As a file format, "lzma" usually refers to either the legacy
[LZMA_ALONE file format](http://svn.python.org/projects/external/xz-5.0.3/doc/lzma-file-format.txt)
or the newer [XZ file format](http://tukaani.org/xz/xz-file-format-1.0.4.txt).
Internally, the LZMA_ALONE format uses the lzma1 filter, while the XZ format
usually uses the lzma2 filter. The LZMA_ALONE format is deprecated and almost
entirely replaced by the XZ format in usage.

### What makes this library different from existing lzma/xz libraries? ###
The main design goals were:

* Provide parallelized compression and decompression of xz files
* Provide seek abilities while reading xz files

As far as the author is aware, neither of these two features are available in
any open source Go library. To accomplish these, the library makes heavy use of
the `liblzma` C implementation.

### Are all xz files seekable? ###
No, the xz file must consist of a series of independently compressed blocks.
If each block size is too small, the compression rate suffers, but the file
provide good random access properties. On the other hand, if each block size is
too large, the compression rate benefits, but the file suffers from poor random
access properties. By default, this library outputs blocks with a 8MiB chunk
size. Thus, in the worst case, a seek will read up to (and discard) 8MiB worth
of data.

The `xz` command-line tool typically outputs xz files with all the data
compressed as a single block. While this library can satisfy the ReadSeeker
interface for this file, seeking to the end of the file is equivalent to
reading the entire file.

### What formats does this library support? ###
Primarily, this library encodes and decodes the XZ file format through the
`goxz/xz` package. However, this library can also encode and decode the
LZMA_ALONE file format through the `goxz/lzma` package. The LZMA_ALONE format is
considered deprecated and use of it is not recommended. It is included in this
library for completeness reasons.

### How does the compression ratio compare to the stock C library? ###
It will be slightly worse. The default uncompressed block-size is 8MiB which
puts an upper limit on how large the dictionary size is and how efficient
compression can get. The disparity is more noticeable when the input data is
highly compressible (where a larger dictionary size benefits most).
Compression performance nearly identical to the C library can be achieved by
simply setting the chunk size to ChunkStream.

## References ##

* [liblzma](http://tukaani.org/xz/) - C library for LZMA/XZ compression
* [compress/lzma](https://code.google.com/p/lzma/) - Pure Go implementation of the LZMA1 filter
* [go-liblzma](https://github.com/remyoudompheng/go-liblzma) - Go bindings for C library
* [pxz](http://jnovy.fedorapeople.org/pxz/) - Parallel compression for XZ
* [pixz](https://github.com/vasi/pixz) - Parallel compression for XZ with indexing