package distromux

import (
	"context"
)

type contextKey struct{}

var distroVarsContextKey = &contextKey{}

func NewDistroVarsContext(parentCtx context.Context, vars DistroVars) context.Context {
	return context.WithValue(parentCtx, distroVarsContextKey, &vars)
}

func DistroVarsFromContext(ctx context.Context) (DistroVars, bool) {
	vars, ok := ctx.Value(distroVarsContextKey).(*DistroVars)
	if !ok {
		return DistroVars{}, false
	}
	return *vars, ok
}
