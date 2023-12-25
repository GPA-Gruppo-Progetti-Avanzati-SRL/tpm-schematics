package schematics

import (
	"fmt"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"regexp"
	"strings"
)

const (
	NameFormattingDasherize = "@dasherize"
)

var schematicsNameRegexp = regexp.MustCompile(`__([a-zA-Z0-9\-]+)(@dasherize|@classify|@camelize|@decamelize|@underscore)?__`)

func ResolveSchematicsName(fn string, props map[string]string) (string, error) {
	matches := schematicsNameRegexp.FindAllSubmatch([]byte(fn), -1)
	for _, m := range matches {
		p := string(m[1])
		mod := string(m[2])

		pv, ok := props[p]
		if !ok {
			return fn, fmt.Errorf("cannot find property %s referenced in name %s", p, fn)
		}

		switch mod {
		case NameFormattingDasherize:
			pv = util.Dasherize(pv)
		}

		fn = strings.ReplaceAll(fn, string(m[0]), pv)
	}

	return fn, nil
}
