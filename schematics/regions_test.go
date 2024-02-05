package schematics_test

import (
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-schematics/schematics"
	"github.com/stretchr/testify/require"
	"testing"
)

var mergerContent = []byte(`
	// @tpm-schematics:start-region("sect2")
    // @tpm-schematics:end-region("sect2")

// More stuff 
	// @tpm-schematics:start-region("sect1")
    // @tpm-schematics:end-region("sect1")
`)

var mergedContent = []byte(`
// @tpm-schematics:start-region("sect1")
fofff
    foff
    // @tpm-schematics:end-region("sect1")

	// @tpm-schematics:start-region("sect2")
    // @tpm-schematics:end-region("sect2")
`)

func TestReadRegionsFromBuffer(t *testing.T) {
	m, err := schematics.ReadRegionsFromBuffer(mergedContent)
	require.NoError(t, err)

	for n, v := range m {
		t.Log(n, v)
	}
}

func TestMergeRegions(t *testing.T) {
	data, err := schematics.RecoverRegions(mergedContent, mergerContent)
	require.NoError(t, err)

	t.Log(string(data))
}
