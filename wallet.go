package wallet

import (
	"bytes"
	"crypto/sha256"
	"crypto/elliptic"
	"crypto/ecdsa"
	"crypto/rand"
	"log"
	"os"
	"fmt"
	"io/ioutil"
	"encoding/gob"
	"golang.org/x/crypto/ripemd160"
	"go_code/A_golang_blockchain/base58"

)
const version = byte(0x00)
const walletFile = "wallet.dat"
const addressChecksumLen = 4 //对校验位一般取4位

//创建一个钱包结构体,钱包里面只装公钥和私钥
type Wallet struct {
	PrivateKey 		ecdsa.PrivateKey
	PublicKey 		[]byte
}

//实例化一个钱包
func NewWallet() *Wallet {
	//生成秘钥对
	private , public := newKeyPair()
	wallet := &Wallet{private,public}
	return wallet
}
//生成密钥对函数
func newKeyPair() (ecdsa.PrivateKey,[]byte) {
	//返回一个实现了P-256的曲线
	curve := elliptic.P256()
	//通过椭圆曲线 随机生成一个私钥
	private ,err := ecdsa.GenerateKey(curve,rand.Reader)
	if err != nil {
		log.Panic(err)
	}
	pubKey := append(private.PublicKey.X.Bytes(),private.PublicKey.Y.Bytes()...)

	return *private,pubKey
}

//生成一个地址
func (w Wallet) GetAddress() []byte {
	//调用公钥哈希函数，实现RIPEMD160(SHA256(Public Key))
	pubKeyHash := HashPubKey(w.PublicKey)
	//存储version和公钥哈希的切片
	versionedPayload := append([]byte{version},pubKeyHash...)
	//调用checksum函数，对上面的切片进行双重哈希后，取出哈希后的切片的前面部分作为检验位的值
	checksum := checksum(versionedPayload)
	//把校验位加到上面切片后面
	fullPayload := append(versionedPayload,checksum...)
	//通过base58编码上述切片得到地址
	address := base58.Base58Encode(fullPayload)

	return address
}
 
//公钥哈希函数，实现RIPEMD160(SHA256(Public Key))
func HashPubKey(pubKey []byte) []byte {
	//先hash公钥
	publicSHA256 := sha256.Sum256(pubKey)
	//对公钥哈希值做 ripemd160运算
	RIPEMD160Hasher := ripemd160.New()
	_,err := RIPEMD160Hasher.Write(publicSHA256[:])
	if err != nil {
		log.Panic(err)
	}
	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)

	return publicRIPEMD160
}
//校验位checksum,双重哈希运算
func checksum(payload []byte) []byte {
	//下面双重哈希payload，在调用中，所引用的payload为（version + Pub Key Hash）
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	//addressChecksumLen代表保留校验位长度
	return secondSHA[:addressChecksumLen]  
}

//判断输入的地址是否有效,主要是检查后面的校验位是否正确
func ValidateAddress(address string) bool {
	//解码base58编码过的地址
	pubKeyHash := base58.Base58Decode([]byte(address))
	//拆分pubKeyHash,pubKeyHash组成形式为：(一个字节的version) + (Public key hash) + (Checksum) 
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1:len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checksum(append([]byte{version},pubKeyHash...))
	//比较拆分出的校验位与计算出的目标校验位是否相等
	return bytes.Compare(actualChecksum,targetChecksum) == 0
}

//创建一个钱包集合的结构体
type Wallets struct {
	Wallets map[string]*Wallet
}

// 实例化一个钱包集合，
func NewWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)
	err := wallets.LoadFromFile()

	return &wallets, err
}

// 将 Wallet 添加进 Wallets
func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := fmt.Sprintf("%s", wallet.GetAddress())
	ws.Wallets[address] = wallet
	return address
}

// 得到存储在wallets里的地址
func (ws *Wallets) GetAddresses() []string {
	var addresses []string
	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}
	return addresses
}
// 通过地址返回出钱包
func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

// 从文件中加载钱包s
func (ws *Wallets) LoadFromFile() error {
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}
	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}
	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}
	ws.Wallets = wallets.Wallets
	return nil
}

// 将钱包s保存到文件
func (ws Wallets) SaveToFile() {
	var content bytes.Buffer
	gob.Register(elliptic.P256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}
	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}
