include bin/build/make/help.mak
include bin/build/make/go.mak
include bin/build/make/git.mak

# Run all package benchmarks. Set benchtime=<duration-or-count> to override the default 100x.
benchmarks:
	@$(MAKE) benchtime=$(or $(benchtime),100x) benchmark

# Run bounded fuzz smoke tests. Set fuzztime=<duration> to override the default 1s per target.
fuzz-smoke: map-fuzz value-fuzz pool-fuzz group-fuzz worker-fuzz

map-fuzz:
	@$(MAKE) package=. name=FuzzMapStringIntOperations fuzztime=$(or $(fuzztime),1s) fuzz
	@$(MAKE) package=. name=FuzzMapNilInterfaceRoundTrip fuzztime=$(or $(fuzztime),1s) fuzz

value-fuzz:
	@$(MAKE) package=. name=FuzzValueIntOperations fuzztime=$(or $(fuzztime),1s) fuzz

pool-fuzz:
	@$(MAKE) package=. name=FuzzBufferPoolCopyAndReset fuzztime=$(or $(fuzztime),1s) fuzz

group-fuzz:
	@$(MAKE) package=. name=FuzzErrorsGroupJoinOrder fuzztime=$(or $(fuzztime),1s) fuzz

worker-fuzz:
	@$(MAKE) package=. name=FuzzWorkerTryScheduleCapacity fuzztime=$(or $(fuzztime),1s) fuzz
