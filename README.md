# Parallelized XZ Compression Library for Go #

## Introduction ##
*INCOMPLETE PROJECT*

## Theory ##
*To be continued*

## Results ##
*To be continued*

## Frequently asked questions ##

### Does this support the legacy lzma format? ###
The compressor will always produce the modern xz file format. The decompressor
can read the legacy lzma file format, but will not attempt to parallelize the
decompression of it.

### Why bother parallelizing decompression? ###
In files where the compression ratio is high, parallel decompression doesn't
achieve much since the consumer of the output likely can't keep up with the
decompressed output. However, experience has shown that a large file often has
sections of highly compressible data, sections of poorly compressible data, and
everything in between. When decompressing a poorly compressible section, the
process is often bounded by the CPU performance of a single core. It is in these
areas that parallel decompression can still provide improvement.

### Why not use the xz-index for parallel decompression? ###
Ideally, yes, since this is what the index field was built for. As such, it
allows random access decompression, and thus parallelized decompression.
The problem with the index is that it is located at the end of a xz-stream.
This implies that the decompressor must be able to seek to the end of the file
to read the index and then seek back to the individual xz-blocks. Since one of
the primary applications of this library was for parallelized decompression in a
streamed manner (where seek is not an available operation), this simply was not
an acceptable solution.

### Why search for the xz-streams instead of the xz-blocks? ###
The individual blocks lack a distinct magic number making it hard to accurately
and easily identify them. On the other hand, the streams have a very distinct
magic number sequence making it extremely easy to search for them and then
speculatively decompress from that point on. Experience has shown that the
number of false positive identifications was relatively low.

Secondly, even if it were easy to identify the individual blocks, in order to
decompress the block requires some information from the stream header (such as
the check type). It becomes tricky to find the stream header that is also
associated with a given block.

### Can this library decompress any xz files in parallel? ###
No, the xz file must be created by concatenating a series of xz-streams.
Even if each stream consists of multiple independently compressed blocks,
the decompressor will not take advantage of the index field to random access
the individual blocks.

### How does the compression ratio compare to the stock C library? ###
It will be slightly worse. The default uncompressed block-size is 1MiB which
puts an upper limit on how large the dictionary size is and how efficient
compression can get. The disparity is more noticeable when the input data is
highly compressible (where a larger dictionary size benefits most).
Compression identical to the C library can be achieved by simply setting the
number of worker routines to 0.

## References ##

* [liblzma](http://tukaani.org/xz/) - Standard C library for lzma compression
* [go-liblzma](https://github.com/remyoudompheng/go-liblzma) - Go bindings for C library
* [pxz](http://jnovy.fedorapeople.org/pxz/) - Parallel compression for xz
