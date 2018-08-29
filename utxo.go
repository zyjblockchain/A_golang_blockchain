package utxo

import (
	"go_code/A_golang_blockchain/transaction"
	"encoding/hex"
	"github.com/boltdb/bolt"
	"go_code/A_golang_blockchain/blockchain"
	"go_code/A_golang_blockchain/block"
	"log"
)
const utxoBucket = "chainstate"

//创建一个结构体，代表UTXO集
type UTXOSet struct {
	Blockchain *blockchain.Blockchain
}

//构建UTXO集的索引并存储在数据库的bucket中
func (u UTXOSet) Reindex() {
	//调用区块链中的数据库,这里的Db()是格式工厂Blockchain结构体中的字段db
	db := u.Blockchain.Db()
	//桶名
	bucketName := []byte(utxoBucket)
	//对数据库进行读写操作
	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(bucketName) //因为我们是要哦重新建一个桶，所以如果原来的数据库中有相同名字的桶，则删除
		if err != nil && err != bolt.ErrBucketNotFound {
			log.Panic(err)
		}
		//创建新桶
		_,err = tx.CreateBucket(bucketName)
		if err != nil {
			log.Panic(err)
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	//返回链上所有未花费交易中的交易输出
	UTXO := u.Blockchain.FindUTXO()

	//把未花费交易中的交易输出集合写入桶中
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)

		//写入键值对
		for txID,outs := range UTXO {
			key,err := hex.DecodeString(txID)
			if err != nil {
				log.Panic(err)
			}
			err = b.Put(key,outs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
} 

//查询并返回被用于这次花费的输出，找到的输出的总额要刚好大于要花费的输入额
func (u UTXOSet) FindSpendableOutputs(pubkeyHash []byte,amount int) (int,map[string][]int) {
	//存储找到的未花费输出集合
	unspentOutputs := make(map[string][]int)
	//记录找到的未花费输出中累加的值
	accumulated := 0
	db := u.Blockchain.Db()

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		//声明一个游标，类似于我们之前构造的迭代器
		c := b.Cursor()

		//用游标来遍历这个桶里的数据,这个桶里装的是链上所有的未花费输出集合
		for k,v := c.First(); k != nil; k,v =c.Next() {
			txID := hex.EncodeToString(k)
			outs := transaction.DeserializeOutputs(v)

			for outIdx,out := range outs.Outputs {
				if out.IsLockedWithKey(pubkeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID],outIdx)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return accumulated,unspentOutputs
}

//查询对应的地址的未花费输出
func (u UTXOSet) FindUTXO(pubKeyHash []byte) []transaction.TXOutput {
	var UTXOs []transaction.TXOutput
	db := u.Blockchain.Db()

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k,v := c.First();k != nil;k,v = c.Next() {
			outs := transaction.DeserializeOutputs(v)

			for _,out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs,out)
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return UTXOs
}

//当区块链中的区块增加后，要同步更新UTXO集,这里引入的区块为新加入的区块。
func (u UTXOSet) Update(block *block.Block) {
	db := u.Blockchain.Db()

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))

		for _,tx := range block.Transactions {
			if tx.IsCoinbase() == false {
				for _,vin := range tx.Vin {
					//实例化结构体TXOutputs
					updatedOuts := transaction.TXOutputs{}
					outsBytes := b.Get(vin.Txid)
					outs := transaction.DeserializeOutputs(outsBytes)

					for outIdx,out := range outs.Outputs {
						if outIdx != vin.Vout {
							updatedOuts.Outputs = append(updatedOuts.Outputs,out)
						}
					}
					if len(updatedOuts.Outputs) == 0 {
						err := b.Delete(vin.Txid)
						if err != nil  {
							log.Panic(err)
						}
					}else{
						err := b.Put(vin.Txid,updatedOuts.Serialize())
						if err != nil {
							log.Panic(err)
						}
					}
				}
			}
			newOutputs := transaction.TXOutputs{}
			for _,out := range tx.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs,out)
			}

			err := b.Put(tx.ID,newOutputs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}
//返回UTXO集中的交易数
func (u UTXOSet) CountTransactions() int {
	db := u.Blockchain.Db() 
	counter := 0

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()

		for k,_ := c.First(); k != nil; k,_ = c.Next() {
			counter++
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return counter
}
