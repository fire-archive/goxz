
# Parallelized xz Compression Library for Go #

## Introduction ##

## Theory ##

## Frequently asked questions ##

### Does this support the legacy lzma format? ###
The compressor will always produce the modern xz file format. The decompressor can read the legacy lzma file format, but will not attempt to parallelize the decompression of it.

### Why bother parallelizing decompression? ###
In files where the compression ratio is high, parallel decompression doesn't achieve much since the consumer of the output likely can't keep up with the decompressed output. However, experience has shown that a large file often has sections of highly compressible data, sections of poorly compressible data, and everything in between. When decompressing a poorly compressible section, the process is often bounded by the CPU performance of a single core. It is in these areas that parallel decompression can still provide improvement.

### Why not use the xz-index for parallel decompression? ###
Ideally, yes, since this is what the index field was built for. As such, it allows random access decompression, and thus parallelized decompression. The problem with the index is that it is located at the end of a xz-stream. What this implies is that, the decompressor must be able to seek to the end of the file to read the index and then seek back to the individual xz-blocks. Since one of the primary applications of this library was for parallelized decompression in a streamed manner (where seek is not an available operation), this simply was not an acceptable solution.

### Why search for the xz-streams instead of the xz-blocks? ###
The individual blocks lack a distinct magic number making it hard to accurately and easily identify them. On the other hand, the streams have a very distinct magic number sequence making it extremely easy to search for them and then speculatively decompress from that point on. Experience has shown that the number of false positive identifications was relatively low.

### Can this library decompress all xz files in parallel? ###
No, the xz file must be created by concatenating a series of xz-streams. Even if each stream consists of multiple independently compressed blocks, the decompressor will not take advantage of the index field to random access the individual blocks.

## References ##

* https://github.com/remyoudompheng/go-liblzma
