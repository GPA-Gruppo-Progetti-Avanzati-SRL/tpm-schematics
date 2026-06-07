package schematics_test

import (
	"embed"
	"testing"
	"text/template"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-schematics/schematics"
	"github.com/stretchr/testify/require"
)

//go:embed example-templates/*
var exampleTemplates embed.FS

func TestGetSource(t *testing.T) {
	metadata := map[string]interface{}{
		"name": "myName",
	}

	model := map[string]interface{}{
		"name": "myName",
	}

	fMap := template.FuncMap{
		"camelize": func(s string) string {
			return util.Camelize(s)
		},
		"dasherize": func(s string) string {
			return util.Dasherize(s)
		},
		"classify": func(s string) string {
			return util.Classify(s)
		},
		"decamelize": func(s string) string {
			return util.Decamelize(s)
		},
		"underscore": func(s string) string {
			return util.Underscore(s)
		},
	}

	_, err := schematics.GetSource(
		exampleTemplates, "example-templates",
		schematics.SourceWithModel(model),
		schematics.SourceWithFuncMap(fMap),
		schematics.SourceWithMetadata(metadata),
	)
	require.NoError(t, err)

}

func TestApply(t *testing.T) {
	metadata := map[string]interface{}{
		"name": "myName-in-metadata",
	}

	model := map[string]interface{}{
		"name": "myName-in-model",
	}

	fMap := template.FuncMap{
		"camelize": func(s string) string {
			return util.Camelize(s)
		},
		"dasherize": func(s string) string {
			return util.Dasherize(s)
		},
		"classify": func(s string) string {
			return util.Classify(s)
		},
		"decamelize": func(s string) string {
			return util.Decamelize(s)
		},
		"underscore": func(s string) string {
			return util.Underscore(s)
		},
	}

	src, err := schematics.GetSource(
		exampleTemplates, "example-templates",
		schematics.SourceWithModel(model),
		schematics.SourceWithFuncMap(fMap),
		schematics.SourceWithMetadata(metadata),
	)
	require.NoError(t, err)

	err = schematics.Apply(
		src,
		schematics.WithFilesystemWriter("/Users/marioa.imperato/tmp/test-sch"),
		schematics.WithApplyDefaultConflictMode(schematics.ConflictModeBackup), schematics.WithDeleteOtherFiles("(.yml)|(.yaml)$"))
	require.NoError(t, err)
}
