include bin/build/make/go.mak
include bin/build/make/git.mak

# Run all the benchmarks.
benchmarks: bytes-benchmarks

bytes-benchmarks:
	@make package=bytes benchmark
