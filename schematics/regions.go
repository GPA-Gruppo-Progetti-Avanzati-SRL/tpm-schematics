package schematics

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"regexp"
	"strings"
)

func RecoverRegionsOfFile(fromFile string, toContent []byte) ([]byte, error) {

	if !util.FileExists(fromFile) {
		return toContent, nil
	}

	b, err := os.ReadFile(fromFile)
	if err != nil {
		return nil, err
	}

	return RecoverRegions(b, toContent)
}

func RecoverRegions(fromContent []byte, toContent []byte) ([]byte, error) {
	const semLogContext = "schematics::recover-regions"

	regs, err := ReadRegionsFromBuffer(fromContent)
	if err != nil {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	if len(regs) == 0 {
		return toContent, nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(toContent))

	var sb strings.Builder
	for scanner.Scan() {
		l := scanner.Text()
		sb.WriteString(l)
		sb.WriteString("\n")
		regType, regionName, ok := getRegionDemarcation(l)
		if ok && regType == "start-region" {
			if data, ok := regs[regionName]; ok {
				sb.WriteString(data)
			}
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return []byte(sb.String()), nil
}

func ReadRegionsFromBuffer(p []byte) (map[string]string, error) {
	const semLogContext = "schematics::read-regions-from-buffer"
	scanner := bufio.NewScanner(bytes.NewReader(p))

	var m map[string]string

	const OutOfRegion = 0
	const InRegion = 1
	s := 0
	var regionName string
	var currentRegionSize int
	var sb strings.Builder
	var lineno int
	var err error
	for scanner.Scan() {
		l := scanner.Text()

		lineno++
		demarcationType, aName, ok := getRegionDemarcation(l)
		switch s {
		case OutOfRegion:
			if ok {
				if demarcationType == "start-region" {
					regionName = aName
					s = InRegion
				} else {
					err = errors.New("wrong region demarcation")
					log.Error().Err(err).Str("name", aName).Str("type", demarcationType).Int("line", lineno).Msg(semLogContext)
					return nil, err
				}
			}
		case InRegion:
			if ok {
				if demarcationType == "end-region" {
					if currentRegionSize > 0 {
						if m == nil {
							m = make(map[string]string)
						}
						m[regionName] = sb.String()
						sb.Reset()
						currentRegionSize = 0
					}
				} else {
					err = errors.New("wrong region demarcation")
					log.Error().Err(err).Str("name", aName).Str("type", demarcationType).Int("line", lineno).Msg(semLogContext)
					return nil, err
				}
				s = OutOfRegion
			} else {
				sb.WriteString(l)
				sb.WriteString("\n")
				currentRegionSize++
			}
		default:
		}

	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		log.Error().Err(err).Int("line", lineno).Msg(semLogContext)
		return nil, err
	}

	return m, nil
}

var RegionDemarcationRegexp = regexp.MustCompile(`@tpm-schematics:(start-region|end-region)\("([a-zA-Z0-9-_]+)"\)`)

func getRegionDemarcation(l string) (string, string, bool) {
	matches := RegionDemarcationRegexp.FindAllSubmatch([]byte(l), -1)

	for _, m := range matches {
		return string(m[1]), string(m[2]), true
	}

	return "", "", false
}
