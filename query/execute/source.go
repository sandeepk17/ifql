package execute

import (
	"fmt"
	"log"
	"time"

	"github.com/influxdata/ifql/query/plan"
)

type Node interface {
	addTransformation(t Transformation)
}

type Source interface {
	Node
	Run()
}

type CreateSource func(spec plan.ProcedureSpec, id DatasetID, sr StorageReader, now time.Time) Source

var procedureToSource = make(map[plan.ProcedureKind]CreateSource)

func RegisterSource(k plan.ProcedureKind, c CreateSource) {
	if procedureToSource[k] != nil {
		panic(fmt.Errorf("duplicate registration for source with procedure kind %v", k))
	}
	procedureToSource[k] = c
}

// storageSource performs storage reads
type storageSource struct {
	id       DatasetID
	reader   StorageReader
	readSpec ReadSpec
	window   Window
	bounds   Bounds

	ts []Transformation

	currentTime Time
}

func NewStorageSource(id DatasetID, r StorageReader, readSpec ReadSpec, bounds Bounds, w Window, currentTime Time) Source {
	return &storageSource{
		id:          id,
		reader:      r,
		readSpec:    readSpec,
		bounds:      bounds,
		window:      w,
		currentTime: currentTime,
	}
}

func (s *storageSource) addTransformation(t Transformation) {
	s.ts = append(s.ts, t)
}

func (s *storageSource) Run() {
	for blocks, mark, ok := s.Next(); ok; blocks, mark, ok = s.Next() {
		blocks.Do(func(b Block) {
			for _, t := range s.ts {
				t.Process(s.id, b)
				//TODO(nathanielc): Also add mechanism to send UpdateProcessingTime calls, when no data is arriving.
				// This is probably not needed for this source, but other sources should do so.
				t.UpdateProcessingTime(s.id, Now())
			}
		})
		for _, t := range s.ts {
			t.UpdateWatermark(s.id, mark)
		}
	}
	for _, t := range s.ts {
		t.Finish(s.id)
	}
}

func (s *storageSource) Next() (BlockIterator, Time, bool) {
	start := s.currentTime - Time(s.window.Period)
	stop := s.currentTime

	s.currentTime = s.currentTime + Time(s.window.Every)
	if stop > s.bounds.Stop {
		return nil, 0, false
	}
	bi, err := s.reader.Read(
		s.readSpec,
		start,
		stop,
	)
	if err != nil {
		log.Println("E!", err)
		return nil, 0, false
	}
	return bi, stop, true
}