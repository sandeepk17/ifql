package functions

import (
	"fmt"
	"math"

	"github.com/influxdata/ifql/ifql"
	"github.com/influxdata/ifql/query"
	"github.com/influxdata/ifql/query/execute"
	"github.com/influxdata/ifql/query/plan"
)

const StddevKind = "stddev"

type StddevOpSpec struct {
}

func init() {
	ifql.RegisterFunction(StddevKind, createStddevOpSpec)
	query.RegisterOpSpec(StddevKind, newStddevOp)
	plan.RegisterProcedureSpec(StddevKind, newStddevProcedure, StddevKind)
	execute.RegisterTransformation(StddevKind, createStddevTransformation)
}
func createStddevOpSpec(args map[string]ifql.Value, ctx ifql.Context) (query.OperationSpec, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf(`stddev function requires no arguments`)
	}

	return new(StddevOpSpec), nil
}

func newStddevOp() query.OperationSpec {
	return new(StddevOpSpec)
}

func (s *StddevOpSpec) Kind() query.OperationKind {
	return StddevKind
}

type StddevProcedureSpec struct {
}

func newStddevProcedure(query.OperationSpec) (plan.ProcedureSpec, error) {
	return new(StddevProcedureSpec), nil
}

func (s *StddevProcedureSpec) Kind() plan.ProcedureKind {
	return StddevKind
}

type StddevAgg struct {
	n, m2, mean float64
}

func createStddevTransformation(id execute.DatasetID, mode execute.AccumulationMode, spec plan.ProcedureSpec, ctx execute.Context) (execute.Transformation, execute.Dataset, error) {
	t, d := execute.NewAggregateTransformationAndDataset(id, mode, ctx.Bounds(), new(StddevAgg))
	return t, d, nil
}

func (a *StddevAgg) Do(vs []float64) {
	var delta, delta2 float64
	for _, v := range vs {
		a.n++
		delta = v - a.mean
		a.mean += delta / a.n
		delta2 = v - a.mean
		a.m2 += delta * delta2
	}
}
func (a *StddevAgg) Value() float64 {
	if a.n < 2 {
		return math.NaN()
	}
	return math.Sqrt(a.m2 / (a.n - 1))
}
func (a *StddevAgg) Reset() {
	a.n = 0
	a.mean = 0
	a.m2 = 0
}