package functions

import (
	"errors"
	"fmt"

	"github.com/influxdata/ifql/ifql"
	"github.com/influxdata/ifql/query"
	"github.com/influxdata/ifql/query/plan"
)

const RangeKind = "range"

type RangeOpSpec struct {
	Start query.Time `json:"start"`
	Stop  query.Time `json:"stop"`
}

func init() {
	ifql.RegisterFunction(RangeKind, createRangeOpSpec)
	query.RegisterOpSpec(RangeKind, newRangeOp)
	plan.RegisterProcedureSpec(RangeKind, newRangeProcedure, RangeKind)
	// TODO register a range transformation. Currently range is only supported if it is pushed down into a select procedure.
	//execute.RegisterTransformation(RangeKind, createRangeTransformation)
}

func createRangeOpSpec(args map[string]ifql.Value) (query.OperationSpec, error) {
	startValue, ok := args["start"]
	if !ok {
		return nil, errors.New(`range function requires argument "start"`)
	}

	spec := new(RangeOpSpec)
	start, err := ifql.ToQueryTime(startValue)
	if err != nil {
		return nil, err
	}
	spec.Start = start

	if stopValue, ok := args["stop"]; ok {
		stop, err := ifql.ToQueryTime(stopValue)
		if err != nil {
			return nil, err
		}
		spec.Stop = stop
	}

	return spec, nil
}

func newRangeOp() query.OperationSpec {
	return new(RangeOpSpec)
}

func (s *RangeOpSpec) Kind() query.OperationKind {
	return RangeKind
}

type RangeProcedureSpec struct {
	Bounds plan.BoundsSpec
}

func newRangeProcedure(qs query.OperationSpec) (plan.ProcedureSpec, error) {
	spec, ok := qs.(*RangeOpSpec)
	if !ok {
		return nil, fmt.Errorf("invalid spec type %T", qs)
	}
	return &RangeProcedureSpec{
		Bounds: plan.BoundsSpec{
			Start: spec.Start,
			Stop:  spec.Stop,
		},
	}, nil
}

func (s *RangeProcedureSpec) Kind() plan.ProcedureKind {
	return RangeKind
}

func (s *RangeProcedureSpec) PushDownRule() plan.PushDownRule {
	return plan.PushDownRule{
		Root:    SelectKind,
		Through: []plan.ProcedureKind{LimitKind, WhereKind},
	}
}
func (s *RangeProcedureSpec) PushDown(root *plan.Procedure) {
	selectSpec := root.Spec.(*SelectProcedureSpec)
	if selectSpec.BoundsSet {
		// TODO: create copy of select spec and set new bounds
		//
		// Example case where this matters
		//    var data = select(database: "mydb")
		//    var past = data.range(start:-2d,stop:-1d)
		//    var current = data.range(start:-1d,stop:now)
	}
	selectSpec.BoundsSet = true
	selectSpec.Bounds = s.Bounds
}