package CLI

import (
	"go_code/A_golang_blockchain/base58"
	"go_code/A_golang_blockchain/transaction"
	"fmt"
	"os"
	"encoding/hex"
	"flag"
	"go_code/A_golang_blockchain/blockchain"
	"go_code/A_golang_blockchain/pow"
	"go_code/A_golang_blockchain/wallet"
	"go_code/A_golang_blockchain/utxo"
	"strconv"
	"log"
)
//首先我们想要拥有这些命令 1.加入区块命令 2.打印区块链命令

//创建一个CLI结构体
type CLI struct {
	//BC *blockchain.Blockchain
}


//加入输入格式错误信息提示
func (cli *CLI) printUsage() {
	fmt.Println("Usage:")
	// fmt.Println("  getbalance -address ADDRESS  得到该地址的余额")
	fmt.Println("  createblockchain -address ADDRESS 创建一条链并且该地址会得到狗头金")
	fmt.Println(" createwallet - 创建一个钱包，里面放着一对秘钥")
	fmt.Println(" getbalance -address ADDRESS  得到该地址的余额")
	fmt.Println("  listaddresses - Lists all addresses from the wallet file")
	fmt.Println("  printchain - 打印链")
	fmt.Println("  reindexutxo - Rebuilds the UTXO set")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT 地址from发送amount的币给地址to")
}

//判断命令行参数，如果没有输入参数则显示提示信息
func (cli *CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

// //加入区块函数调用
// func (cli *CLI) addBlock(data string) {
// 	cli.BC.MineBlock(data)
// 	fmt.Println("成功加入区块...")
// }



//创建一条链
func (cli *CLI) createBlockchain(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}

	bc := blockchain.CreateBlockchain(address)
	defer bc.Db().Close()

	UTXOSet := utxo.UTXOSet{bc}
	UTXOSet.Reindex()
	fmt.Println("Done!")
}

//创建钱包函数
func (cli *CLI) createWallet() {
	wallets, _ := wallet.NewWallets()
	address := wallets.CreateWallet()
	wallets.SaveToFile()
	fmt.Printf("Your new address: %s\n", address)
}

//求账户余额
func (cli *CLI) getBalance(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := blockchain.NewBlockchain()
	UTXOSet := utxo.UTXOSet{bc}
	defer bc.Db().Close()

	balance := 0
	pubKeyHash := base58.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1:len(pubKeyHash)-4] //这里的4是校验位字节数，这里就不在其他包调过来了
	
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)

	//遍历UTXOs中的交易输出out，得到输出字段out.Value,求出余额
	for _,out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of '%s':%d\n",address,balance)
}

//列出地址名单,钱包集合中的地址有哪些
func (cli *CLI) listAddresses() {
	wallets, err := wallet.NewWallets()
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()
	for _, address := range addresses {
		fmt.Println(address)
	}
}

//打印区块链函数调用
func (cli *CLI) printChain() {
	//实例化一条链
	bc := blockchain.NewBlockchain()  //因为已经有了链，不会重新创建链，所以接收的address设置为空
	defer bc.Db().Close()

	//这里需要用到迭代区块链的思想
	//创建一个迭代器
	bci := bc.Iterator()

	for {

		block := bci.Next()	//从顶端区块向前面的区块迭代

		fmt.Printf("------======= 区块 %x ============\n", block.Hash)
		fmt.Printf("时间戳:%v\n",block.Timestamp)
		fmt.Printf("PrevHash:%x\n",block.PrevBlockHash)
		//fmt.Printf("Data:%s\n",block.Data)
		//fmt.Printf("Hash:%x\n",block.Hash)
		//验证当前区块的pow
		pow := pow.NewProofOfWork(block)
		boolen := pow.Validate()
		fmt.Printf("POW is %s\n",strconv.FormatBool(boolen))

		for _,tx := range block.Transactions {
			transaction := (*tx).String()
			fmt.Printf("%s\n",transaction)
		}
		fmt.Printf("\n\n")
		
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}
//查找UTXO集中的交易数
func (cli *CLI) reindexUTXO() {
	bc := blockchain.NewBlockchain()
	UTXOSet := utxo.UTXOSet{bc}
	UTXOSet.Reindex() //在现实中如果能保证自己下载的链节点是完整的，可以忽略。
	count := UTXOSet.CountTransactions()
	fmt.Printf("Done!!! There are %d transactions in the UTXO set.\n", count)
}

//send方法
func (cli *CLI) send(from,to string,amount int) {
	if !wallet.ValidateAddress(from) {
		log.Panic("ERROR: Address is not valid")
	}
	if !wallet.ValidateAddress(to) {
		log.Panic("ERROR: Address is not valid")
	}

	bc := blockchain.NewBlockchain()
	UTXOSet := utxo.UTXOSet{bc}
	defer bc.Db().Close()

	//tx := NewUTXOTransaction(from,to,amount,bc)
	////挖矿奖励的交易，把挖矿的奖励发送给矿工，这里的矿工默认为发送交易的地址
	//cbtx := transaction.NewCoinbaseTX(from,"")

	//挖出一个包含该交易的区块,此时区块还包含了-挖矿奖励的交易
	//bc.MineBlock([]*transaction.Transaction{cbtx,tx})
	tx := NewUTXOTransaction(from, to, amount, &UTXOSet)
	cbTx := transaction.NewCoinbaseTX(from, "")
	txs := []*transaction.Transaction{cbTx, tx}
	newBlock := bc.MineBlock(txs)
	UTXOSet.Update(newBlock)
	fmt.Println("发送成功...")
}

//入口函数 
func (cli *CLI) Run() {
	//判断命令行输入参数的个数，如果没有输入任何参数则打印提示输入参数信息
	cli.validateArgs()
	//实例化flag集合
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	//注册flag标志符
	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	
	switch os.Args[1] {		//os.Args为一个保存输入命令的切片
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	//进入被解析出的命令，进一步操作
	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockchain(*createBlockchainAddress)
	}

	if createWalletCmd.Parsed() {
		cli.createWallet()
	}
	if listAddressesCmd.Parsed() {
		cli.listAddresses()
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}

	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO()
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}
}


//发送币操作,相当于创建一笔未花费输出交易
func NewUTXOTransaction(from,to string,amount int,UTXOSet *utxo.UTXOSet) *transaction.Transaction {
	var inputs []transaction.TXInput
	var outputs []transaction.TXOutput
	//validOutputs是一个存放要用到的未花费输出的交易/输出的map 
	//acc,validOutputs := bc.FindSpendableOutputs(from,amount)
	wallets,err := wallet.NewWallets()
	if err != nil {
		log.Panic(err)
	}
	_wallet := wallets.GetWallet(from)
	pubKeyHash := wallet.HashPubKey(_wallet.PublicKey)
	acc, validOutputs := UTXOSet.FindSpendableOutputs(pubKeyHash, amount)
	if acc < amount {
		log.Panic("ERROR:Not enough tokens...")
	}
	//通过validOutputs里面的数据来放入建立一个输入列表
	for txid,outs := range validOutputs {
		//反序列化得到txID
		txID,err := hex.DecodeString(txid)
		if err != nil {
			log.Panic(err)
		}
		//遍历输出outs切片,得到TXInput里的Vout字段值
		for _,out := range outs {
			//input := transaction.TXInput{txID,out,from}
			input := transaction.TXInput{txID,out,nil,_wallet.PublicKey}
			inputs = append(inputs,input)
		}
	}
	//建立一个输出列表
	//outputs = append(outputs,transaction.TXOutput{amount,to})
	outputs = append(outputs,*transaction.NewTXOutput(amount,to))
	if acc > amount {
		//outputs = append(outputs,transaction.TXOutput{acc - amount,from}) //相当于找零
		outputs = append(outputs,*transaction.NewTXOutput(acc - amount,from)) //相当于找零
	}
	tx := transaction.Transaction{nil,inputs,outputs}
	//tx.SetID()
	tx.ID = tx.Hash()
	UTXOSet.Blockchain.SignTransaction(&tx, _wallet.PrivateKey)

	return &tx
}
