package schematics

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"regexp"
	"strings"
)

const (
	NameFormattingDasherize  = "@dasherize"
	NameFormattingCamelize   = "@camelize"
	NameFormattingDecamelize = "@decamelize"
	NameFormattingClassify   = "@classify"
	NameFormattingUnderscore = "@underscore"
)

var schematicsNameRegexp = regexp.MustCompile(`e\(__([a-zA-Z0-9\-]+)(@dasherize|@classify|@camelize|@decamelize|@underscore)?__\)`)

func ResolveSchematicsName(fn string, props map[string]interface{}) (string, error) {
	matches := schematicsNameRegexp.FindAllSubmatch([]byte(fn), -1)
	for _, m := range matches {
		p := string(m[1])
		mod := string(m[2])

		ipv, ok := props[p]
		if !ok {
			return fn, fmt.Errorf("cannot find property %s referenced in name %s", p, fn)
		}

		pv := fmt.Sprint(ipv)
		switch mod {
		case NameFormattingDasherize:
			pv = util.Dasherize(pv)
		case NameFormattingCamelize:
			pv = util.Camelize(pv)
		case NameFormattingDecamelize:
			pv = util.Decamelize(pv)
		case NameFormattingClassify:
			pv = util.Classify(pv)
		case NameFormattingUnderscore:
			pv = util.Underscore(pv)
		}

		fn = strings.ReplaceAll(fn, string(m[0]), pv)
	}

	return fn, nil
}
