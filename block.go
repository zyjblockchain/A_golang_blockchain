package block

import (
	_"crypto/sha256"
	"encoding/gob"
	"bytes"
	"log"
	"go_code/A_golang_blockchain/transaction"
	"go_code/A_golang_blockchain/merkle_tree"
)
//区块的结构体
type Block struct {
	Timestamp		int64
	Transactions	[]*transaction.Transaction
	PrevBlockHash	[]byte
	Hash 			[]byte
	Nonce			int
}

//区块交易字段的哈希
func (b *Block) HashTransactions() []byte {
	//var txHash [32]byte
	//var txHashes [][]byte
	var transactions  [][]byte

	for _,tx := range b.Transactions {
		//txHashes = append(txHashes,tx.Hash())
		transactions = append(transactions,tx.Serialize())
	}
	//txHash = sha256.Sum256(bytes.Join(txHashes,[]byte{}))
	mTree := merkle_tree.NewMerkleTree(transactions)
	
	//return txHash[:]
	return mTree.RootNode.Data
}

//0.3 实现Block的序列化
func (b *Block) Serialize() []byte {
	//首先定义一个buffer存储序列化后的数据
	var result bytes.Buffer
	//实例化一个序列化实例,结果保存到result中
	encoder := gob.NewEncoder(&result)
	//对区块进行实例化
	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}
	return result.Bytes()
}

//0.3 实现反序列化函数
func DeserializeBlock(d []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(d))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}
	return &block
}
