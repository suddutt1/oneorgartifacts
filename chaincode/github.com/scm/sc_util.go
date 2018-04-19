package main

import (
	"bytes"
	crypto "crypto/x509"
	"encoding/json"
	pem "encoding/pem"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var _SC_LOGGER = shim.NewLogger("SmartContract")

type SmartContract struct {
}
type AbstractInput interface{}

type DataMaskSpec struct {
	NoMask          bool
	MaskedFieldsMap map[string]bool
}
type DataMaskMap struct {
	//ObjectType is the key
	Specification map[string]DataMaskSpec
}

var _MSPWISE_FIELD_MASK_MAP = make(map[string]DataMaskMap)

func (sc *SmartContract) init(stub shim.ChaincodeStubInterface) pb.Response {
	_SC_LOGGER.Info("Inside init method")

	return shim.Success(nil)
}
func (sc *SmartContract) probe(stub shim.ChaincodeStubInterface) pb.Response {

	_SC_LOGGER.Info("Inside probe method")
	ts := time.Now().Format(time.UnixDate)
	output := "{\"status\":\"Success\",\"ts\" : \"" + ts + "\" }"
	_SC_LOGGER.Info("Retuning " + output)
	return shim.Success([]byte(output))
}
func (sc *SmartContract) createAsset(stub shim.ChaincodeStubInterface) pb.Response {
	_, args := stub.GetFunctionAndParameters()
	if len(args) < 2 {
		return shim.Error("Invalid numer of arguments")

	}
	objIDField := args[0]

	isValid, errMsg, assets := sc.ValidateObjectIntegrity(args[1])
	if !isValid {
		return shim.Error(errMsg)
	}
	//Assuming there is only one asset to inserted
	mspID, _, _ := sc.GetIdentity(stub)
	assetToStore := sc.GetMap(assets[0])
	assetToStore["objOwner"] = mspID
	isSaveSuccess, saveErrMsg := sc.ValidateAndInsertObject(stub, assetToStore, objIDField)
	if !isSaveSuccess {
		return shim.Error(saveErrMsg)
	}
	return shim.Success([]byte("Save successful"))
}

//modifyAsset modifies an asset attributes without ownership check
func (sc *SmartContract) modifyAsset(stub shim.ChaincodeStubInterface) pb.Response {
	_, args := stub.GetFunctionAndParameters()
	if len(args) < 2 {
		return shim.Error("Invalid numer of arguments")

	}
	objIDField := args[0]
	modifiedObject := make(map[string]interface{})
	err := json.Unmarshal([]byte(args[1]), &modifiedObject)
	if err != nil {
		return shim.Error("Invalid asset structure provided")
	}
	return sc.ModifyRecord(stub, modifiedObject, objIDField)
}

//transferAssetOwnership transfers the asset to a new owner. The owner should be the MSP on the organization.
func (sc *SmartContract) transferAssetOwnership(stub shim.ChaincodeStubInterface) pb.Response {
	_, args := stub.GetFunctionAndParameters()
	if len(args) < 3 {
		return shim.Error("Invalid numer of arguments")

	}
	objIDField := args[0]
	objID := args[1]
	newOwner := args[2]
	modifiedObject := make(map[string]interface{})
	existingAsset := sc.GetObjectByKey(stub, objID)
	if existingAsset == nil {
		return shim.Error("Asset does not exist. Invlid asset id provided")
	}
	existingAssetDetails := sc.GetMap(existingAsset)
	ownerExistis := sc.IsValidNonEmptyString(existingAssetDetails["objOwner"])
	if !ownerExistis {
		return shim.Error("Asset owner attribute missing for the attribute")
	}
	existingOwner := sc.GetString(existingAssetDetails["objOwner"])
	mspID, _, _ := sc.GetIdentity(stub)
	if existingOwner != mspID {
		return shim.Error("Transaction invoker is not an owner of the asset")
	}
	modifiedObject["objOwner"] = newOwner
	modifiedObject[objIDField] = objID
	return sc.ModifyRecord(stub, modifiedObject, objIDField)
}

//getAssetDetails returns the asset details. Does not perform the ownership check
func (sc *SmartContract) getAssetDetails(stub shim.ChaincodeStubInterface) pb.Response {
	_, args := stub.GetFunctionAndParameters()
	if len(args) < 1 {
		return shim.Error("Invalid numer of arguments")

	}
	objID := args[0]
	asset := sc.GetObjectByKey(stub, objID)
	if asset == nil {
		return shim.Error("Asset does not exist")
	}
	assetJSON, _ := json.Marshal(asset)
	return shim.Success(assetJSON)
}

//ValidateObjectIntegrity Validates the input json array
func (sc *SmartContract) ValidateObjectIntegrity(jsonInput string) (bool, string, []interface{}) {

	var errMsgBuf bytes.Buffer
	var inputObject interface{}
	parsedObjects := make([]interface{}, 0)
	json.Unmarshal([]byte(jsonInput), &inputObject)
	switch inputObject.(type) {
	case []interface{}:
		_SC_LOGGER.Info("Array detected")
		allGood := true

		for index, item := range inputObject.([]interface{}) {
			isGood := sc.CheckObjects(item)
			allGood = allGood && isGood
			if !isGood {
				errMsgBuf.WriteString(fmt.Sprintf("\"Object type missing for %d\",", index))
			}
			parsedObjects = append(parsedObjects, item)
		}

		return allGood, errMsgBuf.String(), parsedObjects
	case interface{}:
		_SC_LOGGER.Info("Object detected")
		isGood := sc.CheckObjects(inputObject)
		if !isGood {
			errMsgBuf.WriteString("Object type missing")
		}
		parsedObjects = append(parsedObjects, inputObject)
		return isGood, errMsgBuf.String(), parsedObjects
	default:
		_SC_LOGGER.Info("Unkown data type")
	}

	return false, "Unkown data type", nil
}

//ValidateAndInsertObject validates the non existance of the object and inserts
func (sc *SmartContract) ValidateAndInsertObject(stub shim.ChaincodeStubInterface, input interface{}, idField string) (bool, string) {
	_SC_LOGGER.Info("ValidateAndInsertObject:Start")
	isSuccess := false
	errMsg := ""
	dataMap, mapOk := input.(map[string]interface{})
	if mapOk == true {
		id, idOk := dataMap[idField].(string)
		if idOk == true && id != "" {
			existingRecord, err := stub.GetState(id)
			_SC_LOGGER.Infof("Existing record %s", string(existingRecord))
			if len(existingRecord) == 0 && err == nil {
				json, _ := json.Marshal(dataMap)
				errSave := stub.PutState(id, json)
				if errSave == nil {
					isSuccess = true
					_SC_LOGGER.Info("Save success")
				} else {
					errMsg = "Not able to save the record in hyperledger"
					_SC_LOGGER.Info("Not able to save")
				}
			} else {
				errMsg = "Id already exists"
				_SC_LOGGER.Info("Id already exists")
			}
		} else {
			errMsg = "Id filed is invalid "
		}
	} else {
		errMsg = "Interface is not a map object"
	}

	return isSuccess, errMsg
}

//ModifyRecord modifies any record generically
func (sc *SmartContract) ModifyRecord(stub shim.ChaincodeStubInterface, modifiedObject map[string]interface{}, id string) pb.Response {
	idField, idOk := modifiedObject[id].(string)
	if idOk == true && idField != "" {
		existingObject := make(map[string]interface{})
		recordBytes, err := stub.GetState(idField)
		if len(recordBytes) > 0 && err == nil {
			_SC_LOGGER.Infof("Record with id %s  does exist", idField)
			json.Unmarshal(recordBytes, &existingObject)
			objectToSave := sc.ModifyObject(existingObject, modifiedObject)
			jsonBytes, _ := json.Marshal(objectToSave)
			jsonBytesPretty, _ := json.MarshalIndent(objectToSave, "", "  ")
			_SC_LOGGER.Infof("Updated record\n%s\n", string(jsonBytesPretty))
			stub.PutState(idField, jsonBytes)
			return shim.Success(jsonBytesPretty)
		}
		_SC_LOGGER.Infof("Record with id %s  does not exist", idField)
		return shim.Error(fmt.Sprintf("Record with id %s  does not exist", idField))
	}
	_SC_LOGGER.Infof("Invalid id field provided")
	return shim.Error("Invalid id field provided")

}

//GetObjectByKey returns data from hyperledger using the key
func (sc *SmartContract) GetObjectByKey(stub shim.ChaincodeStubInterface, id string) interface{} {
	var outputObject interface{}
	recordBytes, err := stub.GetState(id)
	if len(recordBytes) > 0 && err == nil {
		json.Unmarshal(recordBytes, &outputObject)
		return outputObject
	}
	return nil
}

//RetriveRecords based on the selector criteria
func (sc *SmartContract) RetriveRecords(stub shim.ChaincodeStubInterface, criteria, objType string) []map[string]interface{} {

	var selectorString string

	records := make([]map[string]interface{}, 0)
	selectorString = fmt.Sprintf("{\"selector\":%s }", criteria)
	_SC_LOGGER.Info("Query Selector :" + selectorString)
	resultsIterator, _ := stub.GetQueryResult(selectorString)

	for resultsIterator.HasNext() {
		record := make(map[string]interface{})
		recordBytes, _ := resultsIterator.Next()
		err := json.Unmarshal(recordBytes.Value, &record)
		if err != nil {
			_SC_LOGGER.Infof("Unable to unmarshal data retived:: %v", err)
		}
		records = append(records, record)
	}
	return records
}

//CheckObjects checks only the objType attribute. TODO more validations
func (sc *SmartContract) CheckObjects(input interface{}) bool {
	dataMap, ok := input.(map[string]interface{})
	if ok == true && dataMap["objType"] != nil {
		return true
	}
	return false
}

//ModifyObject modifies a record without destrorying the existing fileds except for the arrays
func (sc *SmartContract) ModifyObject(existingRecord, deltaRecord map[string]interface{}) map[string]interface{} {
	for key, value := range deltaRecord {
		switch value.(type) {
		case string:
			existingRecord[key] = value
		case int:
			existingRecord[key] = value
		case []interface{}:
			existingRecord[key] = value
		case interface{}:
			if existingRecord[key] == nil {
				existingRecord[key] = value
			} else {
				deltaRecordMap := value.(map[string]interface{})
				existingRecodMap := existingRecord[key].(map[string]interface{})
				existingRecord[key] = sc.ModifyObject(existingRecodMap, deltaRecordMap)
			}

		}
	}
	return existingRecord
}

func (sc *SmartContract) handleFunctions(stub shim.ChaincodeStubInterface) pb.Response {
	_SC_LOGGER.Info("InsidehandleFunctions")
	function, _ := stub.GetFunctionAndParameters()
	switch function {
	case "probe":
		return sc.probe(stub)
	case "createAsset":
		return sc.createAsset(stub)
	case "modifyAsset":
		return sc.modifyAsset(stub)
	case "transferAssetOwnership":
		return sc.transferAssetOwnership(stub)
	case "getAssetDetails":
		return sc.getAssetDetails(stub)
	}

	return shim.Error("Invalid function provided")
}

//GetIdentity returns the identity of the invoker. Returns the mspId, invoker id from the certificate of the transaction invoker
func (sc *SmartContract) GetIdentity(stub shim.ChaincodeStubInterface) (string, string, error) {
	invokerIDBytes, err := stub.GetCreator()
	if err == nil {
		signingID := &msp.SerializedIdentity{}
		proto.Unmarshal(invokerIDBytes, signingID)
		mspID := signingID.GetMspid()
		_MAIN_LOGGER.Infof("\nMSP ID:%s", mspID)
		idbytes := signingID.GetIdBytes()
		block, _ := pem.Decode(idbytes)
		if block == nil {
			_MAIN_LOGGER.Infof("Expecting a PEM-encoded X509 certificate; PEM block not found")
			return "", "", fmt.Errorf("PEM-encoded X509 certificate in not found in the signature")
		}
		certificate, err := crypto.ParseCertificate(block.Bytes)
		if err != nil {
			_MAIN_LOGGER.Infof("Error in parsing certificate %v", err)
			return "", "", err
		}
		userID := certificate.Subject.CommonName
		_MAIN_LOGGER.Infof("\nCert Subject Common Name %v", userID)
		_MAIN_LOGGER.Infof("\nCert Issuer Common Name %v", certificate.Issuer.CommonName)
		_MAIN_LOGGER.Infof("\nCert Version %v", certificate.Version)
		return mspID, userID, nil

	}
	return "", "", err
}

//GetFloat converts a float to a number
func GetFloat(input interface{}) (float64, bool) {
	dataString, isStringOk := input.(string)
	if isStringOk == true {
		floatValue, pe := strconv.ParseFloat(dataString, 64)
		if pe == nil {
			return floatValue, true
		}
	}
	data, isOk := input.(float64)
	return data, isOk
}
func (sc *SmartContract) IsValidString(input interface{}) bool {
	if input != nil {
		_, ok := input.(string)
		return ok
	}
	return false
}
func (sc *SmartContract) IsValidNonEmptyString(input interface{}) bool {
	if input != nil {
		str, ok := input.(string)
		if ok {
			return strings.TrimSpace(str) != ""
		}
	}
	return false
}
func (sc *SmartContract) IsValidFloat(input interface{}) bool {
	if input != nil {
		_, ok := input.(float64)
		return ok
	}
	return false
}
func (sc *SmartContract) GetString(input interface{}) string {
	if input != nil {
		str, ok := input.(string)
		if ok {
			return str
		}
	}
	return ""
}
func (sc *SmartContract) GetFloat(input interface{}) float64 {
	if input != nil {
		number, ok := input.(float64)
		if ok {
			return number
		}
	}
	return math.NaN()
}
func (sc *SmartContract) GetMap(input interface{}) map[string]interface{} {
	if input != nil {
		output, isMap := input.(map[string]interface{})
		if isMap {
			return output
		}
	}
	return nil
}

func (sc *SmartContract) IsValidMap(input interface{}) bool {
	if input != nil {
		_, isMap := input.(map[string]interface{})
		return isMap
	}
	return false
}
func (sc *SmartContract) IsArray(input interface{}) bool {
	if input != nil {
		_, isArray := input.([]interface{})
		return isArray
	}
	return false
}
func (sc *SmartContract) GetArray(input interface{}) []interface{} {
	if input != nil {
		output, isArry := input.([]interface{})
		if isArry {
			return output
		}
	}
	return nil
}
