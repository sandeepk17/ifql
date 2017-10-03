package plan

import (
	"time"
)

type PlanSpec struct {
	// Now represents the relative currentl time of the plan.
	Now    time.Time
	Bounds BoundsSpec
	// Procedures is a set of all operations
	Procedures map[ProcedureID]*Procedure
	Order      []ProcedureID
	// Results is a list of datasets that are the result of the plan
	Results []ProcedureID
}

func (p *PlanSpec) Do(f func(pr *Procedure)) {
	for _, id := range p.Order {
		f(p.Procedures[id])
	}
}
func (p *PlanSpec) lookup(id ProcedureID) *Procedure {
	return p.Procedures[id]
}

type Planner interface {
	// Plan create a plan from the logical plan and available storage.
	Plan(p *LogicalPlanSpec, s Storage, now time.Time) (*PlanSpec, error)
}

type planner struct {
	plan *PlanSpec
}

func NewPlanner() Planner {
	return new(planner)
}

func (p *planner) Plan(ap *LogicalPlanSpec, s Storage, now time.Time) (*PlanSpec, error) {
	p.plan = &PlanSpec{
		Now:        now,
		Procedures: make(map[ProcedureID]*Procedure, len(ap.Procedures)),
		Order:      make([]ProcedureID, 0, len(ap.Order)),
	}

	// Find the datasets that are results and populate mappings
	ap.Do(func(pr *Procedure) {
		p.plan.Procedures[pr.ID] = pr
		p.plan.Order = append(p.plan.Order, pr.ID)

	})

	// Find Limit+Where+Range+Select to push down time bounds and predicate
	ap.Do(func(pr *Procedure) {
		if pd, ok := pr.Spec.(PushDownProcedureSpec); ok {
			rule := pd.PushDownRule()
			p.pushDownAndSearch(pr, rule, pd.PushDown)
			p.removeProcedure(pr)
		}
		if bounded, ok := pr.Spec.(BoundedProcedureSpec); ok {
			bounds := bounded.TimeBounds()
			p.plan.Bounds = p.plan.Bounds.Union(bounds, now)
		}
	})

	// Now that plan is complete find results
	p.plan.Do(func(pr *Procedure) {
		if len(pr.Children) == 0 {
			p.plan.Results = append(p.plan.Results, pr.ID)
		}
	})

	return p.plan, nil
}

func hasKind(kind ProcedureKind, kinds []ProcedureKind) bool {
	for _, k := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

func (p *planner) pushDownAndSearch(pr *Procedure, rule PushDownRule, do func(parent *Procedure)) {
	for _, parent := range pr.Parents {
		pp := p.plan.Procedures[parent]
		pk := pp.Spec.Kind()
		if pk == rule.Root {
			do(pp)
		} else if hasKind(pk, rule.Through) {
			p.pushDownAndSearch(pp, rule, do)
		} else {
			// Cannot push down
			// TODO: create new branch since procedure cannot be pushed down
		}
	}
}

func (p *planner) removeProcedure(pr *Procedure) {
	delete(p.plan.Procedures, pr.ID)
	p.plan.Order = removeID(p.plan.Order, pr.ID)

	for _, id := range pr.Parents {
		parent := p.plan.Procedures[id]
		parent.Children = removeID(parent.Children, pr.ID)
		parent.Children = append(parent.Children, pr.Children...)
	}
	for _, id := range pr.Children {
		child := p.plan.Procedures[id]
		child.Parents = removeID(child.Parents, pr.ID)
		child.Parents = append(child.Parents, pr.Parents...)
	}
}

func removeID(ids []ProcedureID, remove ProcedureID) []ProcedureID {
	filtered := ids[0:0]
	for i, id := range ids {
		if id == remove {
			filtered = append(filtered, ids[0:i]...)
			filtered = append(filtered, ids[i+1:]...)
			break
		}
	}
	return filtered
}
