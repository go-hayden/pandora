package main

import (
	"bytes"
	"errors"
	"strconv"
)

type Tree struct {
	Level        int
	NodeName     string
	Parent       *Tree
	Reference    []*Tree
	NotReference []*Tree
	Value        interface{}
	Subtrees     []*Tree
}

func (s *Tree) LevelNodeName() string {
	return strconv.Itoa(s.Level) + "." + s.NodeName
}

func (s *Tree) TreeWithName(name string) *Tree {
	c := make(chan *Tree)
	go DepthFirstTraversalSearchTree(s, name, c)
	foundTree := <-c
	return foundTree
}

func (s *Tree) AppendReference(tree *Tree) {
	if tree == nil {
		return
	}
	if s.Reference == nil {
		s.Reference = make([]*Tree, 0, 5)
	} else {
		for _, aExistTree := range s.Reference {
			if aExistTree == tree {
				return
			}
		}
	}
	s.Reference = append(s.Reference, tree)
}

func (s *Tree) AppendNotReference(tree *Tree) {
	if tree == nil {
		return
	}
	if s.NotReference == nil {
		s.NotReference = make([]*Tree, 0, 5)
	} else {
		for _, aExistTree := range s.NotReference {
			if aExistTree == tree {
				return
			}
		}
	}
	s.NotReference = append(s.NotReference, tree)
}

func (s *Tree) AppendSubtree(tree *Tree) {
	if tree == nil {
		return
	}
	if s.Subtrees == nil {
		s.Subtrees = make([]*Tree, 0, 5)
	} else {
		for _, aExistTree := range s.Subtrees {
			if aExistTree == tree {
				return
			}
		}
	}
	s.Subtrees = append(s.Subtrees, tree)
}

func DepthFirstTraversalSearchTree(tree *Tree, nodeName string, c chan *Tree) {
	if tree == nil {
		c <- nil
		return
	}
	if tree.NodeName == nodeName {
		c <- tree
		return
	}
	l := len(tree.Subtrees)
	if l == 0 {
		c <- nil
		return
	}
	nextChan := make(chan *Tree)
	for _, aSubtree := range tree.Subtrees {
		go DepthFirstTraversalSearchTree(aSubtree, nodeName, nextChan)
	}
	var foundTree *Tree
	for i := 0; i < l; i++ {
		tmpTree := <-nextChan
		if tmpTree != nil {
			foundTree = tmpTree
		}
	}
	c <- foundTree
}

func DepthFirstTraversalTree(tree *Tree, f func(aTree *Tree)) {
	if f == nil || tree == nil {
		return
	}
	f(tree)
	for _, subtree := range tree.Subtrees {
		DepthFirstTraversalTree(subtree, f)
	}
}

func NewTreeWithConnectedGraph(graph *ConnectedGraph) (*Tree, error) {
	if graph == nil || len(graph.NodeMap) == 0 {
		return nil, nil
	}
	grapMap := make(map[string]INode)
	for key, val := range graph.NodeMap {
		grapMap[key] = val
	}
	aRootTree := new(Tree)
	aRootTree.NodeName = graph.RootNode
	aRootTree.Level = 1
	preTreeNodes := make([]*Tree, 1, 1)
	preTreeNodes[0] = aRootTree
	level := aRootTree.Level
	for len(preTreeNodes) > 0 {
		level++
		currentLevelTreeNodes := make([]*Tree, 0, 5)
		for _, aParentTree := range preTreeNodes {
			aINode, ok := grapMap[aParentTree.NodeName]
			if !ok {
				aRootTree.AppendNotReference(aParentTree)
				continue
			}
			delete(grapMap, aParentTree.NodeName)

			refNodes := aINode.ReferenceNodes()
			if len(refNodes) == 0 {
				continue
			}

			for _, item := range refNodes {
				exitsTree := aRootTree.TreeWithName(item)
				if exitsTree != nil {
					aParentTree.AppendReference(exitsTree)
					continue
				}
				aNewTree := new(Tree)
				aNewTree.Level = level
				aNewTree.Parent = aParentTree
				aNewTree.NodeName = item
				currentLevelTreeNodes = append(currentLevelTreeNodes, aNewTree)
				aParentTree.AppendSubtree(aNewTree)
			}
		}
		preTreeNodes = currentLevelTreeNodes
	}
	if len(grapMap) > 0 {
		buffer := new(bytes.Buffer)
		for key, _ := range grapMap {
			buffer.WriteString("[" + key + "] ")
		}
		return nil, errors.New("输入的图为非连通图，未联通的节点: " + buffer.String())
	}

	return aRootTree, nil
}
