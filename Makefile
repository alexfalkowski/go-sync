fuzztime ?= 1000x

include bin/build/make/help.mak
include bin/build/make/go.mak
include bin/build/make/git.mak
include bin/build/make/claude.mak
include bin/build/make/codex.mak

# Run all package benchmarks. Set benchtime=<duration-or-count> to override the default 100x.
benchmarks:
	@$(MAKE) benchtime=$(or $(benchtime),100x) benchmark

# Run bounded fuzz tests. Set fuzztime=<duration-or-count> to override the default.
fuzzes: map-fuzz value-fuzz pool-fuzz group-fuzz worker-fuzz

map-fuzz:
	@$(MAKE) package=. name=FuzzMapStringIntOperations fuzz
	@$(MAKE) package=. name=FuzzMapNilInterfaceRoundTrip fuzz

value-fuzz:
	@$(MAKE) package=. name=FuzzValueIntOperations fuzz

pool-fuzz:
	@$(MAKE) package=. name=FuzzBufferPoolCopyAndReset fuzz

group-fuzz:
	@$(MAKE) package=. name=FuzzErrorsGroupJoinOrder fuzz

worker-fuzz:
	@$(MAKE) package=. name=FuzzWorkerTryScheduleCapacity fuzz
