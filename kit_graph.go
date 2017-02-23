package main

import "strings"
import "errors"
import "strconv"
import "encoding/json"

type INode interface {
	ReferenceNodes() []string
}

type ConnectedGraph struct {
	RootNode string
	NodeMap  map[string]INode
}

func ConnectedGraphs(origin map[string]INode) []*ConnectedGraph {
	if len(origin) == 0 {
		return nil
	}
	res := make([]*ConnectedGraph, 0, 5)
	newOrigin := origin
	for newOrigin != nil && len(newOrigin) > 0 {
		var aConnectedGraph *ConnectedGraph
		aConnectedGraph, newOrigin = MaxCnnectedGraph(newOrigin)
		if aConnectedGraph != nil {
			res = append(res, aConnectedGraph)
		}
	}
	return res
}

func MaxCnnectedGraph(origin map[string]INode) (*ConnectedGraph, map[string]INode) {
	var max map[string]bool
	var root string
	for nodeName, _ := range origin {
		ergodic := make(map[string]bool)
		ergodic[nodeName] = false
		for {
			l := len(ergodic)
			count := 0
			for ergodicNodeName, isErgodic := range ergodic {
				if isErgodic {
					count++
					continue
				}
				if ref, ok := origin[ergodicNodeName]; ok {
					nodes := ref.ReferenceNodes()
					for _, item := range nodes {
						if item == ergodicNodeName {
							continue
						}
						if _, ok := ergodic[item]; !ok {
							ergodic[item] = false
						}
					}
				}
				ergodic[ergodicNodeName] = true
			}
			if l == count && l == len(ergodic) {
				break
			}
		}
		if len(ergodic) > len(max) {
			max = ergodic
			root = nodeName
		}
	}
	maxCnnectedGraph := make(map[string]INode)
	otherGraph := make(map[string]INode)
	for key, val := range origin {
		if _, ok := max[key]; ok {
			maxCnnectedGraph[key] = val
		} else {
			otherGraph[key] = val
		}
	}
	aConnectedGraph := new(ConnectedGraph)
	aConnectedGraph.RootNode = root
	aConnectedGraph.NodeMap = maxCnnectedGraph
	return aConnectedGraph, otherGraph
}

type Graph struct {
	Nodes []*GraphNode `json:"nodes,omitempty"`
	Edges []*GraphEdge `json:"edges,omitempty"`
	Name  string       `json:"-,omitempty"`

	currentIndex int                   `json:"-,omitempty"`
	nodesMap     map[string]*GraphNode `json:"-,omitempty"`
}

func (s *Graph) HasNode() bool {
	return len(s.Nodes) > 0
}

func (s *Graph) initIfNeed() {
	if s.Nodes == nil {
		s.Nodes = make([]*GraphNode, 0, 20)
	}
	if s.Edges == nil {
		s.Edges = make([]*GraphEdge, 0, 20)
	}
	if s.nodesMap == nil {
		s.nodesMap = make(map[string]*GraphNode)
	}
}

func (s *Graph) AppendNode(name, description, color, fontColor string, fontSize int) error {
	s.initIfNeed()
	if _, ok := s.nodesMap[name]; ok {
		return errors.New("已存在名为" + name + "的节点!")
	}
	idx := s.NextIndex()
	if name == "" {
		name = "Node "
		for {
			name += strconv.Itoa(idx)
			if _, ok := s.nodesMap[name]; !ok {
				break
			}
		}
	}
	aNode := NewGraphNode(name, description, color, fontColor, fontSize)
	aNode.Index = idx
	s.nodesMap[name] = aNode
	s.Nodes = append(s.Nodes, aNode)
	return nil
}

func (s *Graph) AppendEdge(name string, from, to, arrows, color string, dashes bool) error {
	if from == "" || to == "" {
		return errors.New("from和to指定的节点名称不能为空!")
	}
	s.initIfNeed()

	var fromInt, toInt int
	if aNode, ok := s.nodesMap[from]; ok {
		fromInt = aNode.Index
	} else {
		return errors.New("找不到from指定的节点!")
	}
	if aNode, ok := s.nodesMap[to]; ok {
		toInt = aNode.Index
	} else {
		return errors.New("找不到to指定的节点!")
	}
	aEdge := NewGraphEdge(name, fromInt, toInt, arrows, color, dashes)
	s.Edges = append(s.Edges, aEdge)
	return nil
}

func (s *Graph) CheckNode(nodeName string) bool {
	_, ok := s.nodesMap[nodeName]
	return ok
}

func (s *Graph) NextIndex() int {
	s.currentIndex++
	return s.currentIndex
}

// Layer Interface Impl
func (s *Graph) Title() string {
	return s.Name
}

func (s *Graph) Bytes() ([]byte, error) {
	return json.Marshal(s)
}

type GraphNode struct {
	Name        string     `json:"label,omitempty"`
	Description string     `json:"title,omitempty"`
	Index       int        `json:"id,omitempty"`
	Color       string     `json:"color,omitempty"`
	Font        *GraphFont `json:"font,omitempty"`
}

func NewGraphNode(name, description, color, fontColor string, fontSize int) *GraphNode {
	aNode := new(GraphNode)
	aNode.Name = name
	aNode.Description = description
	if color == "" || !strings.HasPrefix(color, "#") {
		aNode.Color = _COLOR_GRAPH_DEFAULT
	} else {
		aNode.Color = color
	}
	aFont := new(GraphFont)
	if fontColor == "" || !strings.HasPrefix(fontColor, "#") {
		aFont.Color = _COLOR_GRAPH_FONT_DEFAULT
	} else {
		aFont.Color = fontColor
	}
	if fontSize < 1 {
		aFont.Size = _FONT_GRAPH_DEFAULT
	} else {
		aFont.Size = fontSize
	}
	aNode.Font = aFont
	return aNode
}

type GraphEdge struct {
	Name   string `json:"label,omitempty"`
	From   int    `json:"from,omitempty"`
	To     int    `json:"to,omitempty"`
	Arrows string `json:"arrows,omitempty"`
	Color  string `json:"color,omitempty"`
	Dashes bool   `json:"dashes,omitempty"`
}

func NewGraphEdge(naem string, from, to int, arrows, color string, dashes bool) *GraphEdge {
	aEdge := new(GraphEdge)
	aEdge.Name = naem
	aEdge.From = from
	aEdge.To = to
	aEdge.Arrows = arrows
	if color == "" || !strings.HasPrefix(color, "#") {
		aEdge.Color = _COLOR_GRAPH_DEFAULT
	} else {
		aEdge.Color = color
	}
	aEdge.Dashes = dashes
	return aEdge
}

type GraphFont struct {
	Size  int    `json:"size,omitempty"`
	Color string `json:"color,omitempty"`
}
