package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"bytes"

	"github.com/go-hayden-base/fs"
)

type Layer interface {
	Title() string
	Bytes() ([]byte, error)
}

type Canvas struct {
	layers          []Layer
	root            string
	originHtmlPath  string
	originCSSPaths  []string
	originJSPaths   []string
	destRelCSSPaths []string
	destRelJSPaths  []string
	hasWriteCommon  bool
}

// ** Canvas Impl **
func (s *Canvas) Append(layer Layer) {
	s.initIfNeed()
	if layer != nil {
		s.layers = append(s.layers, layer)
	}
}

func (s *Canvas) Output() (string, error) {
	if len(s.layers) == 0 {
		return "没有要绘制的图层", nil
	}

	if e := s.writeCommonIfNeed(); e != nil {
		return "", e
	}

	if !fs.FileExists(s.originHtmlPath) {
		return "", errors.New("模板文件不存在！")
	}

	l := len(s.layers)
	buffers := make([]*bytes.Buffer, l, l)
	for i := 0; i < l; i++ {
		buffers[i] = new(bytes.Buffer)
	}

	var err error
	fs.ReadLine(s.originHtmlPath, func(line string, finished bool, err error, stop *bool) {
		lineType, index := parseTemplateLine(line)
		for idx, aBuffer := range buffers {
			newLine := line
			switch lineType {
			case _TAG_TEMPLATE_GRAPH_TYPE_JS:
				newLine = s.destRelJSPathString(newLine[:index])
			case _TAG_TEMPLATE_GRAPH_TYPE_CSS:
				newLine = s.destRelCSSPathString(newLine[:index])
			case _TAG_TEMPLATE_GRAPH_TYPE_TITLE:
				aLayer := s.layers[idx]
				newLine = strings.Replace(newLine, _TAG_TEMPLATE_GRAPH_TITLE, aLayer.Title(), -1)
			case _TAG_TEMPLATE_GRAPH_TYPE_DATA:
				aLayer := s.layers[idx]
				if b, e := aLayer.Bytes(); e != nil {
					err = e
					*stop = true
					break
				} else {
					newLine = strings.Replace(newLine, _TAG_TEMPLATE_GRAPH_DATA, string(b), -1)
				}
			}
			aBuffer.WriteString(newLine)
			aBuffer.WriteString("\n")
		}
	})
	if err != nil {
		return "", err
	}
	bufferMsg := new(bytes.Buffer)
	for idx, aBuffer := range buffers {
		aLayer := s.layers[idx]
		fileName := aLayer.Title()
		if fileName == "" {
			fileName = string(RandomString(32, KC_RAND_KIND_ALL)) + ".html"
		} else if !strings.HasSuffix(fileName, ".html") {
			fileName += ".html"
		}
		filePath := path.Join(s.root, fileName)
		if e := WriteFile(filePath, aBuffer.Bytes(), true, os.ModePerm); e != nil {
			bufferMsg.WriteString("输出关系图失败: " + filePath + " 原因: " + e.Error())
		} else {
			bufferMsg.WriteString("输出关系图: " + filePath)
		}
		bufferMsg.WriteString("\n")
	}
	return bufferMsg.String(), nil
}

func (s *Canvas) destRelJSPathString(prefix string) string {
	var buffer bytes.Buffer
	for _, item := range s.destRelJSPaths {
		buffer.WriteString(prefix + `<script type="text/javascript" src="` + item + `"></script>`)
		buffer.WriteString("\n")
	}
	return buffer.String()
}

func (s *Canvas) destRelCSSPathString(prefix string) string {
	var buffer bytes.Buffer
	for _, item := range s.destRelCSSPaths {
		buffer.WriteString(prefix + `<link href="` + item + `" rel="stylesheet" type="text/css" />`)
		buffer.WriteString("\n")
	}
	return buffer.String()
}

func parseTemplateLine(line string) (int, int) {
	if idx := strings.Index(line, _TAG_TEMPLATE_GRAPH_TITLE); idx > -1 {
		return _TAG_TEMPLATE_GRAPH_TYPE_TITLE, idx
	} else if idx := strings.Index(line, _TAG_TEMPLATE_GRAPH_JS); idx > -1 {
		return _TAG_TEMPLATE_GRAPH_TYPE_JS, idx
	} else if idx := strings.Index(line, _TAG_TEMPLATE_GRAPH_CSS); idx > -1 {
		return _TAG_TEMPLATE_GRAPH_TYPE_CSS, idx
	} else if idx := strings.Index(line, _TAG_TEMPLATE_GRAPH_DATA); idx > -1 {
		return _TAG_TEMPLATE_GRAPH_TYPE_DATA, idx
	}
	return _TAG_TEMPLATE_GRAPH_TYPE_NORMAL, 0
}

func (s *Canvas) writeCommonIfNeed() error {
	s.initIfNeed()
	if s.hasWriteCommon {
		return nil
	}
	if s.root == "" {
		return errors.New("未设置输出目录!")
	}
	for _, p := range s.originCSSPaths {
		if b, e := ioutil.ReadFile(p); e != nil {
			return e
		} else {
			baseName := path.Base(p)
			out := path.Join(s.root, "css", baseName)
			outRel := path.Join("./css", baseName)
			if e = WriteFile(out, b, true, os.ModePerm); e != nil {
				return nil
			}
			s.destRelCSSPaths = append(s.destRelCSSPaths, outRel)
		}
	}
	for _, p := range s.originJSPaths {
		if b, e := ioutil.ReadFile(p); e != nil {
			return e
		} else {
			baseName := path.Base(p)
			out := path.Join(s.root, "js", baseName)
			outRel := path.Join("./js", baseName)
			if e = WriteFile(out, b, true, os.ModePerm); e != nil {
				return nil
			}
			s.destRelJSPaths = append(s.destRelJSPaths, outRel)
		}
	}
	s.hasWriteCommon = true
	return nil
}

func (s *Canvas) initIfNeed() {
	if s.layers == nil {
		s.layers = make([]Layer, 0, 5)
	}
	if s.destRelCSSPaths == nil {
		s.destRelCSSPaths = make([]string, 0, 5)
	}
	if s.destRelJSPaths == nil {
		s.destRelJSPaths = make([]string, 0, 5)
	}
}

func NewCanvas(outputRoot, htmlPath string, CSSPaths, JSPaths []string) *Canvas {
	aCanvas := new(Canvas)
	aCanvas.initIfNeed()
	aCanvas.root = outputRoot
	aCanvas.originHtmlPath = htmlPath
	aCanvas.originCSSPaths = CSSPaths
	aCanvas.originJSPaths = JSPaths
	return aCanvas
}
