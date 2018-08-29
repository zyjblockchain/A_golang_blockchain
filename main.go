package main

import (
	"go_code/A_golang_blockchain/CLI"
)

	func main() {
		// bc := Blockchain.NewBlockchain()
		// defer bc.Db().Close()
	
		cli := CLI.CLI{}
		cli.Run()
	
	}
