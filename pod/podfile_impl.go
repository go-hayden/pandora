package pod

import (
	"bytes"
	"errors"

	"strings"

	"github.com/go-hayden-base/fs"
)

func (s *Podfile) Read() error {
	if len(s.FilePath) == 0 || !fs.FileExists(s.FilePath) {
		return errors.New("请正确设置Podfile文件路径！")
	}

	s.Targets = make(map[string]map[string]*PFDependence)
	bufferHeader := new(bytes.Buffer)
	bufferAddition := new(bytes.Buffer)
	headerEnd := false
	inTarget := false
	currentTarget := "*"
	fs.ReadLine(s.FilePath, func(line string, finished bool, err error, stop *bool) {
		if len(strings.TrimSpace(line)) == 0 || IsSpecNote(line) {
			return
		}

		tgt, ok := CheckTarget(line)
		if ok {
			headerEnd = true
			inTarget = true
			currentTarget = tgt
			_, ok := s.Targets[tgt]
			if !ok {
				s.Targets[tgt] = make(map[string]*PFDependence)
			}
			return
		}

		d, v, t, ok := CheckPodDep(line)
		if ok {
			headerEnd = true
			pfd := new(PFDependence)
			pfd.Type = t
			if t == "v" {
				pfd.Version = v
			} else {
				pfd.SpecURI = v
			}
			m, ok := s.Targets[currentTarget]
			if !ok {
				m = make(map[string]*PFDependence)
			}

			m[d] = pfd
			s.Targets[currentTarget] = m
			return
		}

		if IsEnd(line) && inTarget {
			inTarget = false
			currentTarget = "*"
			return
		}

		if !headerEnd {
			bufferHeader.WriteString(line + "\n")
		}

		if headerEnd && !inTarget {
			bufferAddition.WriteString(line + "\n")
		}
	})
	s.Header = bufferHeader.String()
	s.AdditionalCode = bufferAddition.String()
	return nil
}

func (s *Podfile) Print() {
	println(s.Header)
	for kt, tgt := range s.Targets {
		println(kt)
		for kd, dps := range tgt {
			println(kd, dps.Type, dps.Version, dps.SpecURI)
		}
	}
	println(s.AdditionalCode)
}
