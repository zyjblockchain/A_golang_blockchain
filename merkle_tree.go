package merkle_tree

import (
	"crypto/sha256"

)

//创建结构体
type MerkleTree struct {
	RootNode *MerkleNode
}

type MerkleNode struct {
	Left 	*MerkleNode
	Right 	*MerkleNode
	Data 	[]byte
}

//创建一个新的节点
func NewMerkleNode(left,right *MerkleNode,data []byte) *MerkleNode {
	mNode := MerkleNode{}

	if left == nil && right == nil {
		//叶子节点
		hash := sha256.Sum256(data)
		mNode.Data = hash[:]
	} else {
		prevHashes := append(left.Data,right.Data...)
		hash := sha256.Sum256(prevHashes)
		mNode.Data = hash[:]
	}

	mNode.Left = left
	mNode.Right = right

	return &mNode
}

//生成一颗新树
func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode

	//输入的交易个数如果是单数的话，就复制最后一个，成为复数
	if len(data) % 2 != 0 {
		data = append(data,data[len(data) - 1])
	}

	//通过数据生成叶子节点
	for _,datum := range data {
		node := NewMerkleNode(nil,nil,datum)
		nodes = append(nodes,*node)
	}
	
	//循环一层一层的生成节点，知道到最上面的根节点为止
	for i := 0; i < len(data)/2; i++ {
		var newLevel []MerkleNode

		for j := 0; j < len(nodes); j += 2 {
			node := NewMerkleNode(&nodes[j],&nodes[j+1],nil)
			newLevel = append(newLevel,*node)
		}

		nodes = newLevel
	}

	mTree := MerkleTree{&nodes[0]}

	return &mTree

}
