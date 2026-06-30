package sync_test

import (
	"bytes"
	"testing"

	"github.com/alexfalkowski/go-sync/internal/test"
)

func FuzzMapStringIntOperations(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 0, 1, 2})
	f.Add([]byte{1, 0, 1, 2})
	f.Add([]byte{2, 0, 1, 2, 2, 0, 3, 4})
	f.Add([]byte{5, 0, 1, 2})
	f.Add([]byte{6, 0, 1, 2, 7, 0, 2, 0})
	f.Add([]byte{8})

	f.Fuzz(func(t *testing.T, data []byte) {
		test.FuzzMapStringIntOperations(t, data)
	})
}

func FuzzMapNilInterfaceRoundTrip(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 0, 0})
	f.Add([]byte{0, 1, 0})
	f.Add([]byte{0, 0, 1})
	f.Add([]byte{2, 0, 0, 5, 0, 0})
	f.Add([]byte{6, 1, 1, 3, 1, 0})

	f.Fuzz(func(t *testing.T, data []byte) {
		test.FuzzMapNilInterfaceRoundTrip(t, data)
	})
}

func FuzzValueIntOperations(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 1, 2})
	f.Add([]byte{1, 1, 2})
	f.Add([]byte{2, 1, 2})
	f.Add([]byte{3, 1, 2, 1, 3, 4})

	f.Fuzz(func(t *testing.T, data []byte) {
		test.FuzzValueIntOperations(t, data)
	})
}

func FuzzBufferPoolCopyAndReset(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("hello"))
	f.Add([]byte{0, 1, 2, 3, 255})
	f.Add(bytes.Repeat([]byte("a"), 1024))

	f.Fuzz(func(t *testing.T, data []byte) {
		test.FuzzBufferPoolCopyAndReset(t, data)
	})
}

func FuzzErrorsGroupJoinOrder(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{1})
	f.Add([]byte{0, 1, 0, 1})
	f.Add([]byte{1, 1, 1})

	f.Fuzz(func(t *testing.T, data []byte) {
		test.FuzzErrorsGroupJoinOrder(t, data)
	})
}

func FuzzWorkerTryScheduleCapacity(f *testing.F) {
	f.Add(0, 0)
	f.Add(0, 3)
	f.Add(1, 3)
	f.Add(4, 16)

	f.Fuzz(func(t *testing.T, limitRaw int, attemptsRaw int) {
		test.FuzzWorkerTryScheduleCapacity(t, limitRaw, attemptsRaw)
	})
}
