package schematics_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"tpm-schematics/schematics"
)

type InputWanted struct {
	input  string
	wanted string
	param  int
}

func TestResolveSchematicsName(t *testing.T) {

	a := assert.New(t)

	var s []InputWanted

	// Decamelize
	s = []InputWanted{
		{input: "e(__name@dasherize__).component.html.template", wanted: "pippo.component.html.template"},
		{input: "e(__name2@dasherize__).component.html.template", wanted: "duffy-duck.component.html.template"},
	}

	props := map[string]interface{}{
		"name":  "pippo",
		"name2": "duffyDuck",
	}

	for _, iw := range s {
		n1, err := schematics.ResolveSchematicsName(iw.input, props)
		require.NoError(t, err)

		fmt.Printf("%s --> %s\n", iw.input, n1)
		a.Equal(iw.wanted, n1)
	}

}
