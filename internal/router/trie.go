package router

type trieNode struct {
	children map[byte]*trieNode
	routes   []*Route
}

func newTrieNode() *trieNode {
	return &trieNode{children: make(map[byte]*trieNode)}
}

func (n *trieNode) insert(prefix string, rt *Route) {
	node := n
	for i := 0; i < len(prefix); i++ {
		c := prefix[i]
		child, ok := node.children[c]
		if !ok {
			child = newTrieNode()
			node.children[c] = child
		}
		node = child
	}
	node.routes = append(node.routes, rt)
}

func (n *trieNode) walk(path string, visit func(*Route)) {
	node := n
	for i := 0; i < len(path); i++ {
		child, ok := node.children[path[i]]
		if !ok {
			return
		}
		node = child
		for _, rt := range node.routes {
			visit(rt)
		}
	}
}
