package schematics

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util"
	"github.com/GPA-Gruppo-Progetti-Avanzati-SRL/tpm-common/util/fileutil"
	"github.com/rs/zerolog/log"
)

const outOfRegion = "out-of-region"
const inRegion = "in-region"
const inRegionSkipContent = "in-region-skip-content"

type RegionInfo struct {
	Name    string
	Content string
	Size    int
}

func RecoverRegionsOfFile(fromFile string, toContent []byte) ([]byte, error) {

	if !fileutil.FileExists(fromFile) {
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

	scanner := bufio.NewReader(bytes.NewReader(toContent))

	status := outOfRegion
	var currentRegionName string

	lineno := 0
	l, err := util.BufoReaderReadLineAsString(scanner, lineno+1, 0)
	var sb strings.Builder
	for err == nil {
		lineno++

		demarcationType, regionName, isDemarcationLine := getRegionDemarcation(l)
		switch status {
		case outOfRegion:
			if isDemarcationLine {
				if demarcationType == "start-region" {
					sb.WriteString(l)
					sb.WriteString("\n")
					currentRegionName = regionName
					if regionInfo, ok := regs[regionName]; ok {
						if regionInfo.Size != 0 {
							sb.WriteString(regionInfo.Content)
							status = inRegionSkipContent
						} else {
							status = inRegion
						}
					} else {
						status = inRegion
					}
				} else {
					err = errors.New("wrong region demarcation")
					log.Error().Err(err).Str("name", regionName).Str("status", status).Str("type", demarcationType).Int("line", lineno).Msg(semLogContext)
					return nil, err
				}
			} else {
				sb.WriteString(l)
				sb.WriteString("\n")
			}
		case inRegion:
			if isDemarcationLine {
				if demarcationType == "end-region" && regionName == currentRegionName {
					sb.WriteString(l)
					sb.WriteString("\n")
					status = outOfRegion
					currentRegionName = ""
				} else {
					err = errors.New("wrong region demarcation")
					log.Error().Err(err).Str("name", regionName).Str("status", status).Str("type", demarcationType).Int("line", lineno).Msg(semLogContext)
					return nil, err
				}
			} else {
				sb.WriteString(l)
				sb.WriteString("\n")
			}
		case inRegionSkipContent:
			if isDemarcationLine {
				if demarcationType == "end-region" && regionName == currentRegionName {
					sb.WriteString(l)
					sb.WriteString("\n")
					status = outOfRegion
					currentRegionName = ""
				} else {
					err = errors.New("wrong region demarcation")
					log.Error().Err(err).Str("name", regionName).Str("status", status).Str("type", demarcationType).Int("line", lineno).Msg(semLogContext)
					return nil, err
				}
			}
		}

		l, err = util.BufoReaderReadLineAsString(scanner, lineno+1, 0)
	}

	if err != io.EOF {
		log.Error().Err(err).Msg(semLogContext)
		return nil, err
	}

	return []byte(sb.String()), nil
}

func ReadRegionsFromBuffer(p []byte) (map[string]RegionInfo, error) {
	const semLogContext = "schematics::read-regions-from-buffer"
	scanner := bufio.NewReader(bytes.NewReader(p))

	var m map[string]RegionInfo

	status := outOfRegion
	var regionName string
	var currentRegionNumLines int
	var sb strings.Builder
	var lineno int
	var err error
	l, err := util.BufoReaderReadLineAsString(scanner, lineno+1, 0)
	for err == nil {
		lineno++
		demarcationType, aName, ok := getRegionDemarcation(l)
		switch status {
		case outOfRegion:
			if ok {
				if demarcationType == "start-region" {
					regionName = aName
					status = inRegion
				} else {
					err = errors.New("wrong region demarcation")
					log.Error().Err(err).Str("name", aName).Str("type", demarcationType).Int("line", lineno).Msg(semLogContext)
					return nil, err
				}
			}
		case inRegion:
			if ok {
				if demarcationType == "end-region" && aName == regionName {
					if m == nil {
						m = make(map[string]RegionInfo)
					}

					rinfo := RegionInfo{
						Name:    regionName,
						Size:    currentRegionNumLines,
						Content: sb.String(),
					}

					m[regionName] = rinfo
					sb.Reset()
					currentRegionNumLines = 0

				} else {
					err = errors.New("wrong region demarcation")
					log.Error().Err(err).Str("name", aName).Str("type", demarcationType).Int("line", lineno).Msg(semLogContext)
					return nil, err
				}
				status = outOfRegion
			} else {
				sb.WriteString(l)
				sb.WriteString("\n")
				currentRegionNumLines++
			}
		default:
		}

		l, err = util.BufoReaderReadLineAsString(scanner, lineno+1, 0)
	}

	if err != io.EOF {
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
