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

// FuzzMapStringIntOperations fuzzes operation scripts because Map must preserve
// sync.Map-like behavior across many operation orders, not only single calls.
func FuzzMapStringIntOperations(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{mapStringIntStore, 0, 1, 0})
	f.Add([]byte{mapStringIntLoad, 0, 0, 0})
	f.Add([]byte{mapStringIntLoadOrStore, 0, 1, 0, mapStringIntLoad, 0, 0, 0})
	f.Add([]byte{mapStringIntSwap, 0, 1, 0})
	f.Add([]byte{mapStringIntCompareAndSwap, 0, 1, 2})
	f.Add([]byte{mapStringIntClear})

	f.Fuzz(func(t *testing.T, script []byte) {
		if len(script) > maxMapStringIntScript {
			script = script[:maxMapStringIntScript]
		}

		scenario := newMapStringIntScenario(t)

		for len(script) > 0 {
			step := readMapStringIntStep(script)
			mapStringIntActions[step.action](scenario, step)
			scenario.requireMatches()
			script = nextFuzzStep(script, mapStringIntStepWidth)
		}
	})
}

// FuzzMapNilInterfaceRoundTrip fuzzes nil interface keys and values because
// Map has to translate untyped nil values stored inside sync.Map back to T.
func FuzzMapNilInterfaceRoundTrip(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{nilInterfaceMapStore, 0, 0})
	f.Add([]byte{nilInterfaceMapStore, 1, 0})
	f.Add([]byte{nilInterfaceMapStore, 0, 1})
	f.Add([]byte{nilInterfaceMapLoadOrStore, 0, 0, nilInterfaceMapLoad, 0, 0})
	f.Add([]byte{nilInterfaceMapSwap, 1, 1, nilInterfaceMapLoadAndDelete, 1, 0})

	f.Fuzz(func(t *testing.T, script []byte) {
		if len(script) > maxNilInterfaceMapScript {
			script = script[:maxNilInterfaceMapScript]
		}

		scenario := newNilInterfaceMapScenario(t)

		for len(script) > 0 {
			step := readNilInterfaceMapStep(script)
			nilInterfaceMapActions[step.action](scenario, step)
			scenario.requireMatches()
			script = nextFuzzStep(script, nilInterfaceMapStepWidth)
		}
	})
}

// FuzzValueIntOperations fuzzes operation scripts because atomic.Value has
// different unset, swap, and compare-and-swap paths that depend on history.
func FuzzValueIntOperations(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{valueIntLoad, 1, 2})
	f.Add([]byte{valueIntStore, 1, 2})
	f.Add([]byte{valueIntSwap, 1, 2})
	f.Add([]byte{valueIntCompareAndSwap, 1, 2, valueIntStore, 1, 0, valueIntCompareAndSwap, 1, 2})

	f.Fuzz(func(t *testing.T, script []byte) {
		if len(script) > maxValueIntScript {
			script = script[:maxValueIntScript]
		}

		value := sync.NewValue[int]()
		var model int
		modelIsSet := false

		for len(script) > 0 {
			step := readValueIntStep(script)

			switch step.action {
			case valueIntLoad:
				require.Equal(t, model, value.Load(), "Load should match model")
			case valueIntStore:
				value.Store(step.current)
				model = step.current
				modelIsSet = true
			case valueIntSwap:
				got := value.Swap(step.current)
				require.Equal(t, model, got, "Swap previous value should match model")
				model = step.current
				modelIsSet = true
			case valueIntCompareAndSwap:
				swapped := value.CompareAndSwap(step.current, step.next)
				want := modelIsSet && model == step.current
				if want {
					model = step.next
				}
				require.Equal(t, want, swapped, "CompareAndSwap result should match model")
			}

			require.Equal(t, model, value.Load(), "Value should match model after step")
			script = nextFuzzStep(script, valueIntStepWidth)
		}
	})
}

// FuzzBufferPoolCopyAndReset fuzzes buffer contents because Copy must detach any
// byte sequence from the pooled buffer before the buffer is reused or mutated.
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
		requireBytesEqual(t, data, copied, "Copy should return buffer contents")

		buffer.Reset()
		_, err = buffer.WriteString("changed")
		require.NoError(t, err)
		requireBytesEqual(t, data, copied, "Copy should not alias the buffer")

		pool.Put(buffer)
		require.Empty(t, buffer.Len(), "Put should reset returned buffer")
		require.Nil(t, pool.Copy(nil), "Copy should return nil for nil buffers")
	})
}

// FuzzErrorsGroupJoinOrder fuzzes nil/error return patterns while forcing
// reverse completion order so Wait proves it joins errors in Go call order.
func FuzzErrorsGroupJoinOrder(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{1})
	f.Add([]byte{0, 1, 0, 1})
	f.Add([]byte{1, 1, 1})

	f.Fuzz(func(t *testing.T, script []byte) {
		if len(script) > maxErrorsGroupScript {
			script = script[:maxErrorsGroupScript]
		}

		var g sync.ErrorsGroup
		releases := make([]chan struct{}, 0, len(script))
		expectedErrors := make([]error, 0, len(script))
		expectedMessages := make([]string, 0, len(script))

		for index, raw := range script {
			release := make(chan struct{})
			releases = append(releases, release)

			var err error
			if raw%2 == 1 {
				err = fmt.Errorf("error %d", index)
				expectedErrors = append(expectedErrors, err)
				expectedMessages = append(expectedMessages, err.Error())
			}

			g.Go(func() error {
				<-release
				return err
			})
		}

		for _, release := range slices.Backward(releases) {
			close(release)
		}

		err := g.Wait()
		if len(expectedErrors) == 0 {
			require.NoError(t, err)
			return
		}

		require.EqualError(t, err, strings.Join(expectedMessages, "\n"))
		for _, expectedErr := range expectedErrors {
			require.ErrorIs(t, err, expectedErr)
		}
	})
}

// FuzzWorkerTryScheduleCapacity fuzzes capacity and attempt counts because
// TrySchedule has important boundaries at zero capacity and exactly-full queues.
func FuzzWorkerTryScheduleCapacity(f *testing.F) {
	f.Add(0, 0)
	f.Add(0, 3)
	f.Add(1, 3)
	f.Add(4, 16)

	f.Fuzz(func(t *testing.T, capacityRaw int, attemptsRaw int) {
		capacity := nonNegativeModulo(capacityRaw, maxFuzzWorkerCapacity)
		attempts := nonNegativeModulo(attemptsRaw, maxFuzzWorkerAttempts)
		worker := sync.NewWorker(uint(capacity))
		release := make(chan struct{})
		var executed sync.Int32

		scheduled := 0
		var schedulingErr error
		for range attempts {
			err := worker.TrySchedule(context.Background(), sync.Hook{
				OnRun: func(context.Context) error {
					executed.Add(1)
					<-release
					return nil
				},
			})

			if scheduled < capacity {
				if err != nil && schedulingErr == nil {
					schedulingErr = err
				}
				if err == nil {
					scheduled++
				}
				continue
			}

			if !errors.Is(err, sync.ErrWorkerFull) && schedulingErr == nil {
				schedulingErr = err
			}
		}

		close(release)
		require.NoError(t, worker.Wait(t.Context()))
		require.NoError(t, schedulingErr)
		require.EqualValues(t, scheduled, executed.Load(), "executed handlers should match successful schedules")
	})
}

const (
	maxMapStringIntScript    = 256
	maxNilInterfaceMapScript = 128
	maxValueIntScript        = 256
	maxErrorsGroupScript     = 32
	maxFuzzWorkerCapacity    = 9
	maxFuzzWorkerAttempts    = 33
	mapStringIntStepWidth    = 4
	nilInterfaceMapStepWidth = 3
	valueIntStepWidth        = 3
)

// Stateful fuzz targets read fixed-width byte scripts: opcode first, then
// operands. Missing operand bytes default to zero so tiny inputs stay valid.
const (
	mapStringIntStore byte = iota
	mapStringIntLoad
	mapStringIntLoadOrStore
	mapStringIntLoadAndDelete
	mapStringIntDelete
	mapStringIntSwap
	mapStringIntCompareAndSwap
	mapStringIntCompareAndDelete
	mapStringIntClear
	mapStringIntActionCount
)

const (
	nilInterfaceMapStore byte = iota
	nilInterfaceMapLoad
	nilInterfaceMapLoadOrStore
	nilInterfaceMapLoadAndDelete
	nilInterfaceMapDelete
	nilInterfaceMapSwap
	nilInterfaceMapClear
	nilInterfaceMapActionCount
)

const (
	valueIntLoad byte = iota
	valueIntStore
	valueIntSwap
	valueIntCompareAndSwap
	valueIntActionCount
)

var mapStringIntActions = [...]func(*mapStringIntScenario, mapStringIntStep){
	(*mapStringIntScenario).store,
	(*mapStringIntScenario).load,
	(*mapStringIntScenario).loadOrStore,
	(*mapStringIntScenario).loadAndDelete,
	(*mapStringIntScenario).delete,
	(*mapStringIntScenario).swap,
	(*mapStringIntScenario).compareAndSwap,
	(*mapStringIntScenario).compareAndDelete,
	(*mapStringIntScenario).clear,
}

var nilInterfaceMapActions = [...]func(*nilInterfaceMapScenario, nilInterfaceMapStep){
	(*nilInterfaceMapScenario).store,
	(*nilInterfaceMapScenario).load,
	(*nilInterfaceMapScenario).loadOrStore,
	(*nilInterfaceMapScenario).loadAndDelete,
	(*nilInterfaceMapScenario).delete,
	(*nilInterfaceMapScenario).swap,
	(*nilInterfaceMapScenario).clear,
}

type mapStringIntStep struct {
	key    string
	value  int
	next   int
	action byte
}

func readMapStringIntStep(script []byte) mapStringIntStep {
	return mapStringIntStep{
		action: script[0] % mapStringIntActionCount,
		key:    string(rune('a' + fuzzByte(script, 1)%8)),
		value:  fuzzInt(script, 2),
		next:   fuzzInt(script, 3),
	}
}

type mapStringIntScenario struct {
	t     *testing.T
	m     *sync.Map[string, int]
	model map[string]int
}

func newMapStringIntScenario(t *testing.T) *mapStringIntScenario {
	t.Helper()

	return &mapStringIntScenario{
		t:     t,
		m:     sync.NewMap[string, int](),
		model: map[string]int{},
	}
}

func (s *mapStringIntScenario) store(step mapStringIntStep) {
	s.m.Store(step.key, step.value)
	s.model[step.key] = step.value
}

func (s *mapStringIntScenario) load(step mapStringIntStep) {
	s.t.Helper()

	got, ok := s.m.Load(step.key)
	want, wantOK := s.model[step.key]
	require.Equal(s.t, wantOK, ok, "Load presence should match model")
	require.Equal(s.t, want, got, "Load value should match model")
}

func (s *mapStringIntScenario) loadOrStore(step mapStringIntStep) {
	s.t.Helper()

	got, loaded := s.m.LoadOrStore(step.key, step.value)
	want, wantLoaded := s.model[step.key]
	if !wantLoaded {
		want = step.value
		s.model[step.key] = step.value
	}
	require.Equal(s.t, wantLoaded, loaded, "LoadOrStore loaded flag should match model")
	require.Equal(s.t, want, got, "LoadOrStore value should match model")
}

func (s *mapStringIntScenario) loadAndDelete(step mapStringIntStep) {
	s.t.Helper()

	got, loaded := s.m.LoadAndDelete(step.key)
	want, wantLoaded := s.model[step.key]
	if wantLoaded {
		delete(s.model, step.key)
	}
	require.Equal(s.t, wantLoaded, loaded, "LoadAndDelete loaded flag should match model")
	require.Equal(s.t, want, got, "LoadAndDelete value should match model")
}

func (s *mapStringIntScenario) delete(step mapStringIntStep) {
	s.m.Delete(step.key)
	delete(s.model, step.key)
}

func (s *mapStringIntScenario) swap(step mapStringIntStep) {
	s.t.Helper()

	got, loaded := s.m.Swap(step.key, step.value)
	want, wantLoaded := s.model[step.key]
	s.model[step.key] = step.value
	require.Equal(s.t, wantLoaded, loaded, "Swap loaded flag should match model")
	require.Equal(s.t, want, got, "Swap previous value should match model")
}

func (s *mapStringIntScenario) compareAndSwap(step mapStringIntStep) {
	s.t.Helper()

	swapped := s.m.CompareAndSwap(step.key, step.value, step.next)
	want := false
	if current, ok := s.model[step.key]; ok && current == step.value {
		s.model[step.key] = step.next
		want = true
	}
	require.Equal(s.t, want, swapped, "CompareAndSwap result should match model")
}

func (s *mapStringIntScenario) compareAndDelete(step mapStringIntStep) {
	s.t.Helper()

	deleted := s.m.CompareAndDelete(step.key, step.value)
	want := false
	if current, ok := s.model[step.key]; ok && current == step.value {
		delete(s.model, step.key)
		want = true
	}
	require.Equal(s.t, want, deleted, "CompareAndDelete result should match model")
}

func (s *mapStringIntScenario) clear(mapStringIntStep) {
	s.m.Clear()
	s.model = map[string]int{}
}

func (s *mapStringIntScenario) requireMatches() {
	s.t.Helper()

	seen := map[string]int{}
	s.m.Range(func(key string, value int) bool {
		seen[key] = value
		return true
	})

	require.Equal(s.t, s.model, seen, "Range should expose the same entries as the model")
	for key, want := range s.model {
		got, ok := s.m.Load(key)
		require.True(s.t, ok, "Load should find model key %q", key)
		require.Equal(s.t, want, got, "Load should return model value for key %q", key)
	}
}

type nilInterfaceMapStep struct {
	key        fmt.Stringer
	value      io.Reader
	keyID      int
	action     byte
	valueIsNil bool
}

func readNilInterfaceMapStep(script []byte) nilInterfaceMapStep {
	keyID := int(fuzzByte(script, 1) % 2)
	valueIsNil := fuzzByte(script, 2)%2 == 0

	return nilInterfaceMapStep{
		action:     script[0] % nilInterfaceMapActionCount,
		key:        nilInterfaceKey(keyID),
		keyID:      keyID,
		value:      nilInterfaceValue(valueIsNil),
		valueIsNil: valueIsNil,
	}
}

type nilInterfaceMapScenario struct {
	t     *testing.T
	m     *sync.Map[fmt.Stringer, io.Reader]
	model map[int]nilInterfaceEntry
}

func newNilInterfaceMapScenario(t *testing.T) *nilInterfaceMapScenario {
	t.Helper()

	return &nilInterfaceMapScenario{
		t:     t,
		m:     sync.NewMap[fmt.Stringer, io.Reader](),
		model: map[int]nilInterfaceEntry{},
	}
}

func (s *nilInterfaceMapScenario) store(step nilInterfaceMapStep) {
	s.m.Store(step.key, step.value)
	s.model[step.keyID] = nilInterfaceEntry{present: true, valueIsNil: step.valueIsNil}
}

func (s *nilInterfaceMapScenario) load(step nilInterfaceMapStep) {
	s.t.Helper()

	got, ok := s.m.Load(step.key)
	want := s.model[step.keyID]
	require.Equal(s.t, want.present, ok, "Load presence should match nil-interface model")
	require.Equal(s.t, !want.present || want.valueIsNil, got == nil, "Load nilness should match model")
}

func (s *nilInterfaceMapScenario) loadOrStore(step nilInterfaceMapStep) {
	s.t.Helper()

	got, loaded := s.m.LoadOrStore(step.key, step.value)
	want, wasPresent := s.model[step.keyID]
	if !wasPresent {
		s.model[step.keyID] = nilInterfaceEntry{present: true, valueIsNil: step.valueIsNil}
		require.False(s.t, loaded, "LoadOrStore should store missing nil-interface key")
		require.Equal(s.t, step.valueIsNil, got == nil, "LoadOrStore stored value nilness should match input")
		return
	}

	require.True(s.t, loaded, "LoadOrStore should load present nil-interface key")
	require.Equal(s.t, want.valueIsNil, got == nil, "LoadOrStore loaded value nilness should match model")
}

func (s *nilInterfaceMapScenario) loadAndDelete(step nilInterfaceMapStep) {
	s.t.Helper()

	got, loaded := s.m.LoadAndDelete(step.key)
	want := s.model[step.keyID]
	if want.present {
		delete(s.model, step.keyID)
	}
	require.Equal(s.t, want.present, loaded, "LoadAndDelete loaded flag should match nil-interface model")
	require.Equal(s.t, !want.present || want.valueIsNil, got == nil, "LoadAndDelete nilness should match model")
}

func (s *nilInterfaceMapScenario) delete(step nilInterfaceMapStep) {
	s.m.Delete(step.key)
	delete(s.model, step.keyID)
}

func (s *nilInterfaceMapScenario) swap(step nilInterfaceMapStep) {
	s.t.Helper()

	got, loaded := s.m.Swap(step.key, step.value)
	want := s.model[step.keyID]
	s.model[step.keyID] = nilInterfaceEntry{present: true, valueIsNil: step.valueIsNil}
	require.Equal(s.t, want.present, loaded, "Swap loaded flag should match nil-interface model")
	require.Equal(s.t, !want.present || want.valueIsNil, got == nil, "Swap previous value nilness should match model")
}

func (s *nilInterfaceMapScenario) clear(nilInterfaceMapStep) {
	s.m.Clear()
	s.model = map[int]nilInterfaceEntry{}
}

func (s *nilInterfaceMapScenario) requireMatches() {
	s.t.Helper()

	seen := map[int]nilInterfaceEntry{}
	s.m.Range(func(key fmt.Stringer, value io.Reader) bool {
		keyID := 1
		if key == nil {
			keyID = 0
		}
		seen[keyID] = nilInterfaceEntry{present: true, valueIsNil: value == nil}
		return true
	})

	require.Equal(s.t, s.model, seen, "Range should expose the same nil-interface entries as the model")
	for id, want := range s.model {
		got, ok := s.m.Load(nilInterfaceKey(id))
		require.Equal(s.t, want.present, ok, "Load presence should match model for key id %d", id)
		require.Equal(s.t, want.valueIsNil, got == nil, "Load nilness should match model for key id %d", id)
	}
}

type valueIntStep struct {
	current int
	next    int
	action  byte
}

func readValueIntStep(script []byte) valueIntStep {
	return valueIntStep{
		action:  script[0] % valueIntActionCount,
		current: fuzzInt(script, 1),
		next:    fuzzInt(script, 2),
	}
}

type nilInterfaceEntry struct {
	present    bool
	valueIsNil bool
}

type fuzzStringer string

func (s fuzzStringer) String() string {
	return string(s)
}

func nextFuzzStep(script []byte, width int) []byte {
	if len(script) <= width {
		return nil
	}
	return script[width:]
}

func fuzzByte(script []byte, index int) byte {
	if index >= len(script) {
		return 0
	}
	return script[index]
}

func fuzzInt(script []byte, index int) int {
	return int(fuzzByte(script, index)%17) - 8
}

func nonNegativeModulo(value int, limit int) int {
	if limit <= 0 {
		return 0
	}
	value %= limit
	if value < 0 {
		return -value
	}
	return value
}

func nilInterfaceKey(id int) fmt.Stringer {
	if id == 0 {
		var key fmt.Stringer
		return key
	}
	return fuzzStringer("key")
}

func nilInterfaceValue(isNil bool) io.Reader {
	if isNil {
		var value io.Reader
		return value
	}
	return strings.NewReader("value")
}

func requireBytesEqual(t *testing.T, want []byte, got []byte, message string) {
	t.Helper()

	if len(want) == 0 {
		require.Empty(t, got, message)
		return
	}
	require.Equal(t, want, got, message)
}
