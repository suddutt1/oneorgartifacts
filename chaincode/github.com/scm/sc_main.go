package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var _MAIN_LOGGER = shim.NewLogger("SmartContractMain")
var CONV_RATIO = 100

// Init initializes chaincode.
func (sc *SmartContract) Init(stub shim.ChaincodeStubInterface) pb.Response {
	_MAIN_LOGGER.Infof("Inside the init method ")

	response := sc.init(stub)
	return response
}

//Invoke is the entry point for any transaction
func (sc *SmartContract) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	return sc.handleFunctions(stub)
}

func main() {
	err := shim.Start(new(SmartContract))
	if err != nil {
		_MAIN_LOGGER.Criticalf("Error starting  chaincode: %v", err)
	}
}
