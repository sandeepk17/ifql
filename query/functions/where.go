package functions

import (
	"fmt"

	"github.com/influxdata/ifql/query"
	"github.com/influxdata/ifql/query/plan"
)

const WhereKind = "where"

type WhereOpSpec struct {
	Exp *query.ExpressionSpec `json:"exp"`
}

func init() {
	query.RegisterOpSpec(WhereKind, newWhereOp)
	plan.RegisterProcedureSpec(WhereKind, newWhereProcedure, WhereKind)
	// TODO register a where transformation. Currently where is only supported if it is pushed down into a select procedure.
	//execute.RegisterTransformation(WhereKind, createWhereTransformation)
}

func newWhereOp() query.OperationSpec {
	return new(WhereOpSpec)
}

func (s *WhereOpSpec) Kind() query.OperationKind {
	return WhereKind
}

type WhereProcedureSpec struct {
	Exp *query.ExpressionSpec
}

func newWhereProcedure(qs query.OperationSpec) (plan.ProcedureSpec, error) {
	spec, ok := qs.(*WhereOpSpec)
	if !ok {
		return nil, fmt.Errorf("invalid spec type %T", qs)
	}

	return &WhereProcedureSpec{
		Exp: spec.Exp,
	}, nil
}

func (s *WhereProcedureSpec) Kind() plan.ProcedureKind {
	return WhereKind
}

func (s *WhereProcedureSpec) PushDownRule() plan.PushDownRule {
	return plan.PushDownRule{
		Root:    SelectKind,
		Through: []plan.ProcedureKind{LimitKind, RangeKind},
	}
}
func (s *WhereProcedureSpec) PushDown(root *plan.Procedure) {
	selectSpec := root.Spec.(*SelectProcedureSpec)
	if selectSpec.WhereSet {
		// TODO: create copy of select spec and set new where expression
	}
	selectSpec.WhereSet = true
	selectSpec.Where = s.Exp.Predicate
}
