package distromux

import (
	"context"
	"testing"

	"github.com/go-test/deep"
)

func TestDistroVarsContext(t *testing.T) {
	vars := DistroVars{}
	vars["foo"] = "bar"

	varsContext := NewDistroVarsContext(context.Background(), vars)
	retrievedVars, ok := DistroVarsFromContext(varsContext)
	if !ok {
		t.Errorf("unable to retrieve distrovars from context")
	}

	if diff := deep.Equal(retrievedVars, vars); len(diff) > 0 {
		t.Errorf("retrieved vars not equal to expected:")
		for _, l := range diff {
			t.Error(l)
		}
	}
}
