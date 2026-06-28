package sync_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
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
		fuzzMapStringIntOperations(t, data)
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
		fuzzMapNilInterfaceRoundTrip(t, data)
	})
}

func FuzzValueIntOperations(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 1, 2})
	f.Add([]byte{1, 1, 2})
	f.Add([]byte{2, 1, 2})
	f.Add([]byte{3, 1, 2, 1, 3, 4})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 256 {
			data = data[:256]
		}

		value := sync.NewValue[int]()
		var model int
		set := false

		for len(data) > 0 {
			op := data[0] % 4
			current := fuzzMapInt(data, 1)
			next := fuzzMapInt(data, 2)

			switch op {
			case 0:
				require.Equal(t, model, value.Load())
			case 1:
				value.Store(current)
				model = current
				set = true
			case 2:
				got := value.Swap(current)
				require.Equal(t, model, got)
				model = current
				set = true
			case 3:
				got := value.CompareAndSwap(current, next)
				want := set && model == current
				if want {
					model = next
				}
				require.Equal(t, want, got)
			}

			require.Equal(t, model, value.Load())
			if len(data) < 3 {
				break
			}
			data = data[3:]
		}
	})
}

func FuzzBufferPoolCopyAndReset(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("hello"))
	f.Add([]byte{0, 1, 2, 3, 255})
	f.Add(bytes.Repeat([]byte("a"), 1024))

	f.Fuzz(func(t *testing.T, data []byte) {
		pool := sync.NewBufferPool()
		buffer := pool.Get()

		_, err := buffer.Write(data)
		require.NoError(t, err)

		copied := pool.Copy(buffer)
		if len(data) == 0 {
			require.Empty(t, copied)
		} else {
			require.Equal(t, data, copied)
		}

		buffer.Reset()
		_, err = buffer.WriteString("changed")
		require.NoError(t, err)
		if len(data) == 0 {
			require.Empty(t, copied)
		} else {
			require.Equal(t, data, copied)
		}

		pool.Put(buffer)
		require.Empty(t, buffer.Len(), "Put should reset returned buffer")
		require.Nil(t, pool.Copy(nil))
	})
}

func FuzzErrorsGroupJoinOrder(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{1})
	f.Add([]byte{0, 1, 0, 1})
	f.Add([]byte{1, 1, 1})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 32 {
			data = data[:32]
		}

		var g sync.ErrorsGroup
		releases := make([]chan struct{}, len(data))
		expected := make([]error, 0, len(data))
		expectedMessages := make([]string, 0, len(data))

		for i, b := range data {
			releases[i] = make(chan struct{})
			var err error
			if b%2 == 1 {
				err = fmt.Errorf("error %d", i)
				expected = append(expected, err)
				expectedMessages = append(expectedMessages, err.Error())
			}

			g.Go(func() error {
				<-releases[i]
				return err
			})
		}

		for _, release := range slices.Backward(releases) {
			close(release)
		}

		err := g.Wait()
		if len(expected) == 0 {
			require.NoError(t, err)
			return
		}

		require.EqualError(t, err, strings.Join(expectedMessages, "\n"))
		for _, expectedErr := range expected {
			require.ErrorIs(t, err, expectedErr)
		}
	})
}

func FuzzWorkerTryScheduleCapacity(f *testing.F) {
	f.Add(0, 0)
	f.Add(0, 3)
	f.Add(1, 3)
	f.Add(4, 16)

	f.Fuzz(func(t *testing.T, limitRaw int, attemptsRaw int) {
		limit := fuzzBoundedInt(limitRaw, 9)
		attempts := fuzzBoundedInt(attemptsRaw, 33)
		worker := sync.NewWorker(uint(limit))
		release := make(chan struct{})
		var executed sync.Int32

		successes := 0
		var unexpected error
		for range attempts {
			err := worker.TrySchedule(context.Background(), sync.Hook{
				OnRun: func(context.Context) error {
					executed.Add(1)
					<-release
					return nil
				},
			})
			if successes < limit {
				if err != nil && unexpected == nil {
					unexpected = err
				}
				successes++
				continue
			}
			if !errors.Is(err, sync.ErrWorkerFull) && unexpected == nil {
				unexpected = err
			}
		}

		close(release)
		worker.Wait()
		require.NoError(t, unexpected)
		require.EqualValues(t, successes, executed.Load())
	})
}

func fuzzMapStringIntOperations(t *testing.T, data []byte) {
	t.Helper()

	if len(data) > 256 {
		data = data[:256]
	}

	m := sync.NewMap[string, int]()
	model := map[string]int{}
	ops := [...]mapStringIntOp{
		mapStringIntStore,
		mapStringIntLoad,
		mapStringIntLoadOrStore,
		mapStringIntLoadAndDelete,
		mapStringIntDelete,
		mapStringIntSwap,
		mapStringIntCompareAndSwap,
		mapStringIntCompareAndDelete,
		mapStringIntClear,
	}

	for len(data) > 0 {
		op := ops[int(data[0])%len(ops)]
		key := fuzzMapKey(data, 1)
		value := fuzzMapInt(data, 2)
		next := fuzzMapInt(data, 3)

		model = op(t, m, model, key, value, next)
		requireMapMatchesModel(t, m, model)
		if len(data) < 4 {
			break
		}
		data = data[4:]
	}
}

type mapStringIntOp func(
	t *testing.T,
	m *sync.Map[string, int],
	model map[string]int,
	key string,
	value int,
	next int,
) map[string]int

func mapStringIntStore(
	t *testing.T,
	m *sync.Map[string, int],
	model map[string]int,
	key string,
	value int,
	_ int,
) map[string]int {
	t.Helper()

	m.Store(key, value)
	model[key] = value
	return model
}

func mapStringIntLoad(
	t *testing.T,
	m *sync.Map[string, int],
	model map[string]int,
	key string,
	_ int,
	_ int,
) map[string]int {
	t.Helper()

	got, ok := m.Load(key)
	want, wantOK := model[key]
	require.Equal(t, wantOK, ok)
	require.Equal(t, want, got)
	return model
}

func mapStringIntLoadOrStore(
	t *testing.T,
	m *sync.Map[string, int],
	model map[string]int,
	key string,
	value int,
	_ int,
) map[string]int {
	t.Helper()

	got, loaded := m.LoadOrStore(key, value)
	want, wantLoaded := model[key]
	if !wantLoaded {
		want = value
		model[key] = value
	}
	require.Equal(t, wantLoaded, loaded)
	require.Equal(t, want, got)
	return model
}

func mapStringIntLoadAndDelete(
	t *testing.T,
	m *sync.Map[string, int],
	model map[string]int,
	key string,
	_ int,
	_ int,
) map[string]int {
	t.Helper()

	got, loaded := m.LoadAndDelete(key)
	want, wantLoaded := model[key]
	if wantLoaded {
		delete(model, key)
	}
	require.Equal(t, wantLoaded, loaded)
	require.Equal(t, want, got)
	return model
}

func mapStringIntDelete(
	t *testing.T,
	m *sync.Map[string, int],
	model map[string]int,
	key string,
	_ int,
	_ int,
) map[string]int {
	t.Helper()

	m.Delete(key)
	delete(model, key)
	return model
}

func mapStringIntSwap(
	t *testing.T,
	m *sync.Map[string, int],
	model map[string]int,
	key string,
	value int,
	_ int,
) map[string]int {
	t.Helper()

	got, loaded := m.Swap(key, value)
	want, wantLoaded := model[key]
	model[key] = value
	require.Equal(t, wantLoaded, loaded)
	require.Equal(t, want, got)
	return model
}

func mapStringIntCompareAndSwap(
	t *testing.T,
	m *sync.Map[string, int],
	model map[string]int,
	key string,
	value int,
	next int,
) map[string]int {
	t.Helper()

	got := m.CompareAndSwap(key, value, next)
	want := false
	if current, ok := model[key]; ok && current == value {
		model[key] = next
		want = true
	}
	require.Equal(t, want, got)
	return model
}

func mapStringIntCompareAndDelete(
	t *testing.T,
	m *sync.Map[string, int],
	model map[string]int,
	key string,
	value int,
	_ int,
) map[string]int {
	t.Helper()

	got := m.CompareAndDelete(key, value)
	want := false
	if current, ok := model[key]; ok && current == value {
		delete(model, key)
		want = true
	}
	require.Equal(t, want, got)
	return model
}

func mapStringIntClear(
	t *testing.T,
	m *sync.Map[string, int],
	_ map[string]int,
	_ string,
	_ int,
	_ int,
) map[string]int {
	t.Helper()

	m.Clear()
	return map[string]int{}
}

func fuzzMapNilInterfaceRoundTrip(t *testing.T, data []byte) {
	t.Helper()

	if len(data) > 128 {
		data = data[:128]
	}

	m := sync.NewMap[fmt.Stringer, io.Reader]()
	model := map[int]nilInterfaceEntry{}
	ops := [...]mapNilInterfaceOp{
		mapNilInterfaceStore,
		mapNilInterfaceLoad,
		mapNilInterfaceLoadOrStore,
		mapNilInterfaceLoadAndDelete,
		mapNilInterfaceDelete,
		mapNilInterfaceSwap,
		mapNilInterfaceClear,
	}

	for len(data) > 0 {
		op := ops[int(data[0])%len(ops)]
		keyID := int(fuzzByte(data, 1) % 2)
		valueNil := fuzzByte(data, 2)%2 == 0
		key := fuzzInterfaceKey(keyID)
		value := fuzzInterfaceValue(valueNil)

		model = op(t, m, model, key, keyID, value, valueNil)
		requireNilInterfaceMapMatchesModel(t, m, model)
		if len(data) < 3 {
			break
		}
		data = data[3:]
	}
}

type mapNilInterfaceOp func(
	t *testing.T,
	m *sync.Map[fmt.Stringer, io.Reader],
	model map[int]nilInterfaceEntry,
	key fmt.Stringer,
	keyID int,
	value io.Reader,
	valueNil bool,
) map[int]nilInterfaceEntry

func mapNilInterfaceStore(
	t *testing.T,
	m *sync.Map[fmt.Stringer, io.Reader],
	model map[int]nilInterfaceEntry,
	key fmt.Stringer,
	keyID int,
	value io.Reader,
	valueNil bool,
) map[int]nilInterfaceEntry {
	t.Helper()

	m.Store(key, value)
	model[keyID] = nilInterfaceEntry{present: true, valueNil: valueNil}
	return model
}

func mapNilInterfaceLoad(
	t *testing.T,
	m *sync.Map[fmt.Stringer, io.Reader],
	model map[int]nilInterfaceEntry,
	key fmt.Stringer,
	keyID int,
	_ io.Reader,
	_ bool,
) map[int]nilInterfaceEntry {
	t.Helper()

	got, ok := m.Load(key)
	want := model[keyID]
	require.Equal(t, want.present, ok)
	require.Equal(t, !want.present || want.valueNil, got == nil)
	return model
}

func mapNilInterfaceLoadOrStore(
	t *testing.T,
	m *sync.Map[fmt.Stringer, io.Reader],
	model map[int]nilInterfaceEntry,
	key fmt.Stringer,
	keyID int,
	value io.Reader,
	valueNil bool,
) map[int]nilInterfaceEntry {
	t.Helper()

	got, loaded := m.LoadOrStore(key, value)
	want, wasPresent := model[keyID]
	if !wasPresent {
		model[keyID] = nilInterfaceEntry{present: true, valueNil: valueNil}
		require.False(t, loaded)
		require.Equal(t, valueNil, got == nil)
		return model
	}

	require.True(t, loaded)
	require.Equal(t, want.valueNil, got == nil)
	return model
}

func mapNilInterfaceLoadAndDelete(
	t *testing.T,
	m *sync.Map[fmt.Stringer, io.Reader],
	model map[int]nilInterfaceEntry,
	key fmt.Stringer,
	keyID int,
	_ io.Reader,
	_ bool,
) map[int]nilInterfaceEntry {
	t.Helper()

	got, loaded := m.LoadAndDelete(key)
	want := model[keyID]
	if want.present {
		delete(model, keyID)
	}
	require.Equal(t, want.present, loaded)
	require.Equal(t, !want.present || want.valueNil, got == nil)
	return model
}

func mapNilInterfaceDelete(
	t *testing.T,
	m *sync.Map[fmt.Stringer, io.Reader],
	model map[int]nilInterfaceEntry,
	key fmt.Stringer,
	keyID int,
	_ io.Reader,
	_ bool,
) map[int]nilInterfaceEntry {
	t.Helper()

	m.Delete(key)
	delete(model, keyID)
	return model
}

func mapNilInterfaceSwap(
	t *testing.T,
	m *sync.Map[fmt.Stringer, io.Reader],
	model map[int]nilInterfaceEntry,
	key fmt.Stringer,
	keyID int,
	value io.Reader,
	valueNil bool,
) map[int]nilInterfaceEntry {
	t.Helper()

	got, loaded := m.Swap(key, value)
	want := model[keyID]
	model[keyID] = nilInterfaceEntry{present: true, valueNil: valueNil}
	require.Equal(t, want.present, loaded)
	require.Equal(t, !want.present || want.valueNil, got == nil)
	return model
}

func mapNilInterfaceClear(
	t *testing.T,
	m *sync.Map[fmt.Stringer, io.Reader],
	_ map[int]nilInterfaceEntry,
	_ fmt.Stringer,
	_ int,
	_ io.Reader,
	_ bool,
) map[int]nilInterfaceEntry {
	t.Helper()

	m.Clear()
	return map[int]nilInterfaceEntry{}
}

func fuzzMapKey(data []byte, index int) string {
	return string(rune('a' + fuzzByte(data, index)%8))
}

func fuzzMapInt(data []byte, index int) int {
	return int(fuzzByte(data, index)%17) - 8
}

func fuzzBoundedInt(value int, limit int) int {
	if limit <= 0 {
		return 0
	}
	value %= limit
	if value < 0 {
		return -value
	}
	return value
}

func fuzzByte(data []byte, index int) byte {
	if index >= len(data) {
		return 0
	}
	return data[index]
}

func requireMapMatchesModel(t *testing.T, m *sync.Map[string, int], model map[string]int) {
	t.Helper()

	seen := map[string]int{}
	m.Range(func(key string, value int) bool {
		seen[key] = value
		return true
	})

	require.Equal(t, model, seen)
	for key, want := range model {
		got, ok := m.Load(key)
		require.True(t, ok)
		require.Equal(t, want, got)
	}
}

type nilInterfaceEntry struct {
	present  bool
	valueNil bool
}

type fuzzStringer string

func (s fuzzStringer) String() string {
	return string(s)
}

func fuzzInterfaceKey(id int) fmt.Stringer {
	if id == 0 {
		var key fmt.Stringer
		return key
	}
	return fuzzStringer("key")
}

func fuzzInterfaceValue(isNil bool) io.Reader {
	if isNil {
		var value io.Reader
		return value
	}
	return strings.NewReader("value")
}

func requireNilInterfaceMapMatchesModel(
	t *testing.T,
	m *sync.Map[fmt.Stringer, io.Reader],
	model map[int]nilInterfaceEntry,
) {
	t.Helper()

	seen := map[int]nilInterfaceEntry{}
	m.Range(func(key fmt.Stringer, value io.Reader) bool {
		keyID := 1
		if key == nil {
			keyID = 0
		}
		seen[keyID] = nilInterfaceEntry{present: true, valueNil: value == nil}
		return true
	})

	require.Equal(t, model, seen)
	for id, want := range model {
		got, ok := m.Load(fuzzInterfaceKey(id))
		require.Equal(t, want.present, ok)
		require.Equal(t, want.valueNil, got == nil)
	}
}
