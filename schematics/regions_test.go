package schematics_test

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-schematics/schematics"
	"github.com/stretchr/testify/require"
	"testing"
)

var newContent = []byte(`
	// @tpm-schematics:start-region("sect1")
line in new content but current has content
    // @tpm-schematics:end-region("sect1")

stuff 01 - out of region 

	// @tpm-schematics:start-region("sect2")
line in new content but current is empty
    // @tpm-schematics:end-region("sect2")

stuff 02 - out of region 

	// @tpm-schematics:start-region("sect3")
line in new content region but current is not existent
    // @tpm-schematics:end-region("sect3")

stuff 03 - out of region 

	// @tpm-schematics:start-region("sect4") 
    // @tpm-schematics:end-region("sect4")

	// @tpm-schematics:start-region("sect5")
    // @tpm-schematics:end-region("sect5")
`)

var currentContent = []byte(`
	// @tpm-schematics:start-region("sect2")
    // @tpm-schematics:end-region("sect2")

	// @tpm-schematics:start-region("sect4")
    // @tpm-schematics:end-region("sect4")

// @tpm-schematics:start-region("sect1")
sect1-merged-content-line1
    sect1-merged-content-line2
    // @tpm-schematics:end-region("sect1")
`)

func TestReadRegionsFromBuffer(t *testing.T) {
	m, err := schematics.ReadRegionsFromBuffer(currentContent)
	require.NoError(t, err)

	for n, v := range m {
		t.Log(n, v.Name, v.Size, v.Content)
	}
}

func TestMergeRegions(t *testing.T) {
	data, err := schematics.RecoverRegions(currentContent, newContent)
	require.NoError(t, err)

	fmt.Println(string(data))
}
