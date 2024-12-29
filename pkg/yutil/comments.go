package yutil

import (
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func SetComments(comments ...string) CommentsSetter {
	var head, line, foot string
	switch len(comments) {
	case 1:
		line = comments[0]
	case 2:
		head = comments[0]
		foot = comments[1]
	case 3:
		head = comments[0]
		line = comments[1]
		foot = comments[2]
	}

	return CommentsSetter{
		HeadComment: head,
		LineComment: line,
		FootComment: foot,
	}
}

type CommentsSetter struct {
	HeadComment string
	LineComment string
	FootComment string
}

func (cs CommentsSetter) Filter(rn *yaml.RNode) (*yaml.RNode, error) {
	if rn == nil {
		return nil, fmt.Errorf("nil rnode")
	}
	if rn.YNode() == nil {
		rn.SetYNode(yaml.MakeNullNode().YNode())
	}
	yn := rn.YNode()
	yn.HeadComment = cs.HeadComment
	yn.LineComment = cs.LineComment
	yn.FootComment = cs.FootComment
	return rn, nil
}

func FixComments(node *yaml.Node) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			k, v := node.Content[i], node.Content[i+1]
			FixComments(v)

			if k.HeadComment != "" && v.HeadComment != "" {
				k.HeadComment += "\n"
			}
			if k.FootComment != "" && v.FootComment != "" {
				v.FootComment += "\n"
			}

			k.HeadComment = k.HeadComment + v.HeadComment
			k.FootComment = v.FootComment + k.FootComment
			v.HeadComment = ""
			v.FootComment = ""
		}
	case yaml.SequenceNode:
		nodes := node.Content
		if len(nodes) < 1 {
			return
		}

		for i := 0; i < len(node.Content)-1; i++ {
			FixComments(nodes[i])

			if nodes[i].FootComment != "" && nodes[i+1].HeadComment != "" {
				nodes[i].FootComment += "\n"
			}

			nodes[i+1].HeadComment = nodes[i].FootComment + nodes[i+1].HeadComment
			nodes[i].FootComment = ""
		}

		last := nodes[len(nodes)-1]
		FixComments(last)

		if node.FootComment != "" && last.FootComment != "" {
			last.FootComment += "\n"
		}

		node.FootComment = last.FootComment + node.FootComment
		last.FootComment = ""
	}
}
