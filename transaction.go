package transaction

import (
	"strings"
	"math/big"
	"crypto/elliptic"
	"encoding/hex"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/rand"
	"encoding/gob"
	"bytes"
	"fmt"
	"log"
	"go_code/A_golang_blockchain/wallet"
	"go_code/A_golang_blockchain/base58"
	
)

var subsidy int = 50  //挖矿奖励
/*创建一个交易的数据结构，交易是由交易ID、交易输入、交易输出组成的,
一个交易有多个输入和多个输出，所以这里的交易输入和输出应该是切片类型的
*/ 
type Transaction struct {
	ID		[]byte
	Vin		[]TXInput
	Vout	[]TXOutput
}

/*
1、每一笔交易的输入都会引用之前交易的一笔或多笔交易输出
2、交易输出保存了输出的值和锁定该输出的信息
3、交易输入保存了引用之前交易输出的交易ID、具体到引用
该交易的第几个输出、能正确解锁引用输出的签名信息
*/
//交易输出
type TXOutput struct {
	Value			int	//输出的值（可以理解为金额）
	//ScriptPubKey	string	// 锁定该输出的脚本（目前还没实现地址，所以还不能锁定该输出为具体哪个地址所有）
	PubkeyHash 		[]byte //存储“哈希”后的公钥，这里的哈希不是单纯的sha256
}
//交易输入
type TXInput struct {
	Txid 		[]byte //引用的之前交易的ID
	Vout		int 	//引用之前交易输出的具体是哪个输出（一个交易中输出一般有很多）
	//ScriptSig	string  // 能解锁引用输出交易的签名脚本（目前还没实现地址，所以本章不能实现此功能）
	Signature 	[]byte //签名脚本
	PubKey 		[]byte // 公钥，这里的公钥是正真的公钥
}

/*
	区块链上存储的交易都是由这些输入输出交易所组成的，
一个输入交易必须引用之前的输出交易，一个输出交易会被之后的输入所引用。
    问题来了，在最开始的区块链上是先有输入还是先有输出喃？
答案是先有输出，因为是区块链的创世区块产生了第一个输出，
这个输出也就是我们常说的挖矿奖励-狗头金，每一个区块都会有一个这样的输出，
这是奖励给矿工的交易输出，这个输出是凭空产生的。
*/
//现在我们来创建一个这样的coinbase挖矿输出
//to 代表此输出奖励给谁，一般都是矿工地址，data是交易附带的信息
func NewCoinbaseTX(to,data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("奖励给 '%s'",to)
	}
	//此交易中的交易输入,没有交易输入信息
	//txin := TXInput{[]byte{},-1,[]byte{},}
	txin := TXInput{[]byte{},-1,nil,[]byte(data)}
	//交易输出,subsidy为奖励矿工的币的数量
	txout := NewTXOutput(subsidy,to)
	//组成交易
	//tx := Transaction{nil,[]TXInput{txin},[]TXOutput{txout}}
	tx := Transaction{nil,[]TXInput{txin},[]TXOutput{*txout}}

	//设置该交易的ID
	//tx.SetID()
	tx.ID = tx.Hash()
	return &tx
}

////设置交易ID，交易ID是序列化tx后再哈希
//func (tx *Transaction) SetID() {
//返回一个序列化后的交易
func (tx Transaction) Serialize() []byte {
	//var hash [32]byte
	var encoder bytes.Buffer

	enc := gob.NewEncoder(&encoder)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	//hash = sha256.Sum256(encoder.Bytes())
	//tx.ID =  hash[:]
	return encoder.Bytes()
}
//返回交易的哈希值
func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

/*
1、每一个区块至少存储一笔coinbase交易，所以我们在区块的字段中把Data字段换成交易。
2、把所有涉及之前Data字段都要换了，比如NewBlock()、GenesisBlock()、pow里的函数
*/

//定义在输入和输出上的锁定和解锁方法，目的是让用户只能花自己所用于地址上的币
//输入上锁定的秘钥,表示能引用的输出是unlockingData
// func (in *TXInput) CanUnlockOutputWith(unlockingData string) bool {
// 	return in.ScriptSig == unlockingData
// }
//方法检查输入是否使用了指定密钥来解锁一个输出
func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := wallet.HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash,pubKeyHash) == 0
}

//输出上的解锁秘钥,表示能被引用的输入是unlockingData
// func (out *TXOutput) CanBeUnlockedWith(unlockingData string) bool {
// 	return out.ScriptPubKey == unlockingData 
// }

//锁定交易输出到固定的地址，代表该输出只能由指定的地址引用
func (out *TXOutput) Lock(address []byte) {
	pubKeyHash := base58.Base58Decode(address)
	pubKeyHash = pubKeyHash[1:len(pubKeyHash)-4]
	out.PubkeyHash = pubKeyHash 
}

//判断输入的公钥"哈希"能否解锁该交易输出
func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PubkeyHash,pubKeyHash) == 0
}
//创建一个新的交易输出
func NewTXOutput(value int,address string) *TXOutput {
	txo := &TXOutput{value,nil}
	txo.Lock([]byte(address))

	return txo
}

//判断是否为coinbase交易
func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

//对交易签名
func (tx *Transaction) Sign(privKey ecdsa.PrivateKey,prevTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	} 

	for _,vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}
	txCopy := tx.TrimmedCopy()

	for inID,vin := range txCopy.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubkeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		r,s,err := ecdsa.Sign(rand.Reader,&privKey,txCopy.ID)
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(),s.Bytes()...)

		tx.Vin[inID].Signature = signature
	}

}

//验证 交易输入的签名
func (tx *Transaction) Verify(prevTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}
	for _,vin := range tx.Vin {
		//遍历输入交易，如果发现输入交易引用的上一交易的ID不存在，则Panic
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}
	txCopy := tx.TrimmedCopy() //修剪后的副本
	curve := elliptic.P256() //椭圆曲线实例

	for inID,vin := range tx.Vin {
		prevTX := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil //双重验证
		txCopy.Vin[inID].PubKey = prevTX.Vout[vin.Vout].PubkeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Vin[inID].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve,&x,&y}
		if ecdsa.Verify(&rawPubKey,txCopy.ID,&r,&s) == false {
			return false
		}
	}
	return true
}

//创建在签名中修剪后的交易副本,之所以要这个副本是因为简化了输入交易本身的签名和公钥
func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _,vin := range tx.Vin {
		inputs = append(inputs,TXInput{vin.Txid,vin.Vout,nil,nil})
	}

	for _,vout := range tx.Vout {
		outputs = append(outputs,TXOutput{vout.Value,vout.PubkeyHash})
	}

	txCopy := Transaction{tx.ID,inputs,outputs}

	return txCopy
}

//把交易转换成我们能正常读的形式
func (tx Transaction) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("--Transaction %x:", tx.ID))
	for i, input := range tx.Vin {
		lines = append(lines, fmt.Sprintf(" -Input %d:", i))
		lines = append(lines, fmt.Sprintf("  TXID: %x", input.Txid))
		lines = append(lines, fmt.Sprintf("  Out:  %d", input.Vout))
		lines = append(lines, fmt.Sprintf("  Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("  PubKey:%x", input.PubKey))
	}
	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf(" -Output %d:", i))
		lines = append(lines, fmt.Sprintf("  Value: %d", output.Value))
		lines = append(lines, fmt.Sprintf("  Script: %x", output.PubkeyHash))
	}
	return strings.Join(lines,"\n")

}

//创建一个结构体，用于表示TXOutput集
type TXOutputs struct {
	Outputs []TXOutput
}
//序列化此集合
func(outs TXOutputs) Serialize() []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(outs)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

//反序列化
func DeserializeOutputs(data []byte) TXOutputs {
	var outputs TXOutputs

	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outputs)
	if err != nil {
		log.Panic(err)
	}

	return outputs
}
