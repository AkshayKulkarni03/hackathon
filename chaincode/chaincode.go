/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

//WARNING - this chaincode's ID is hard-coded in chaincode_example04 to illustrate one way of
//calling chaincode from a chaincode. If this example is modified, chaincode_example04.go has
//to be modified as well with the new ID of chaincode_example02.
//chaincode_example05 show's how chaincode ID can be passed in as a parameter instead of
//hard-coding.

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	//"github.com/op/go-logging"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
	// "github.com/errorpkg"
)

var recType = []string{"USER", "CREATECONTR", "BID", "POSTTRAN", "CLOSECONTRACT", "CANCELCONTRACT"}

//////////////////////////////////////////////////////////////////////////////////////////////////
// The following array holds the list of tables that should be created
// The deploy/init deletes the tables and recreates them every time a deploy is invoked
//////////////////////////////////////////////////////////////////////////////////////////////////
var aucTables = []string{"UserTable", "UserCatTable", "ContractTable", "ContractCatTable", "ContractOpenTable", "BidTable", "BidCatTable",  "BidHistoryTable", "TransTable"}

///////////////////////////////////////////////////////////////////////////////////////
// This creates a record of the Asset (Inventory)
// Includes Description, title, certificate of authenticity or image whatever..idea is to checkin a image and store it
// in encrypted form
// Example:
// Item { 113869, "Flower Urn on a Patio", "Liz Jardine", "10102007", "Original", "Floral", "Acrylic", "15 x 15 in", "sample_9.png","$600", "My Gallery }
///////////////////////////////////////////////////////////////////////////////////////

type ContractObject struct {
	ContractId             string
	Amount                 string
	Duration               string
	BusinessRule           string
	Type                   string
	RequirementDescription string
	Description            string
	Terms                  string
	CreationDate           string
	UserID				   string
	Status                 string //This will have three values OPEN/CLOSED/IN_PROGRESS
	RecType                string
}

/////////////////////////////////////////////////////////////
// Could establish valid UserTypes -
// AH (Auction House)
// TR (Buyer or Seller)
// AP (Appraiser)
// IN (Insurance)
// BK (bank)
// SH (Shipper)
/////////////////////////////////////////////////////////////
type UserObject struct {
	UserID    string
	RecType   string // Type = USER
	Name      string
	UserType  string // Auction House (AH), Bank (BK), Buyer or Seller (TR), Shipper (SH), Appraiser (AP)
	Address   string
	Phone     string
	Email     string
	Bank      string
	AccountNo string
	Rating    string
}

////////////////////////////////////////////////////////////////
//  This is a Bid. Bids are accepted only if an auction is OPEN
////////////////////////////////////////////////////////////////

type Bid struct {
	ContractId string
	RecType    string // BID
	BidNo      string
	UserID     string // ID Of Buyer - to be verified against the Item CurrentOwnerId
	BidPrice   string // BidPrice
	BidTime    string // Time the bid was received
}

/////////////////////////////////////////////////////////////
// POST the transaction after the Auction Completes
// Post an Auction Transaction
// Post an Updated Item Object
// Once an auction request is opened for auctions, a timer is kicked
// off and bids are accepted. When the timer expires, the highest bid
// is selected and converted into a Transaction
// This transaction is a simple view
/////////////////////////////////////////////////////////////

type ItemTransaction struct {
	ConractId         string
	RecType           string // POSTTRAN
	TransactionId     string
	TransType         string // Sale, Buy, Commission
	UserId            string // Buyer or Seller ID
	TransDate         string // Date of Settlement (Buyer or Seller)
	TransactionAmount string // Time of hammer strike - SOLD
	BidNo      		  string
}

func GetNumberOfKeys(tname string) int {
	TableMap := map[string]int{
		"UserTable":        1,
		"ContractTable":    1,
		"UserCatTable":     2,
		"ContractCatTable": 2,
		"ContractOpenTable":2,
		"BidTable":     	1,
		"BidCatTable":     	2,
		"BidHistoryTable":  3,
		"TransTable":       2,
	}
	return TableMap[tname]
}

// ChainCode setup for services
type SimpleChaincode struct {
}

var gopath string
var ccPath string

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// SimpleChaincode - Init Chaincode implementation - The following sequence of transactions can be used to test the Chaincode
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func (t *SimpleChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {

	// TODO - Include all initialization to be complete before Invoke and Query
	// Uses aucTables to delete tables if they exist and re-create them

	//myLogger.Info("[Trade and Auction Application] Init")
	fmt.Println("[Create Contract and R] Init")
	var err error

	for _, val := range aucTables {
		err = stub.DeleteTable(val)
		if err != nil {
			return nil, fmt.Errorf("Init(): DeleteTable of %s  Failed ", val)
		}
		err = InitLedger(stub, val)
		if err != nil {
			return nil, fmt.Errorf("Init(): InitLedger of %s  Failed ", val)
		}
	}

	fmt.Println("Init() Initialization Complete  : ", args)
	return []byte("Init(): Initialization Complete"), nil
}

//////////////////////////////////////////////////////////////
// Invoke Functions based on Function name
// The function name gets resolved to one of the following calls
// during an invoke
//
//////////////////////////////////////////////////////////////
func InvokeFunction(fname string) func(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	InvokeFunc := map[string]func(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error){
		"PostRequest":     PostRequest,
		"SelectBidder":    SelectBidder,
		"PostTransaction": PostTransaction,
		"PostBid":         PostBid,
		"CloseConract":    CloseContract,
	}
	return InvokeFunction[fname]
}

//////////////////////////////////////////////////////////////
// Query Functions based on Function name
//
//////////////////////////////////////////////////////////////
func QueryFunction(fname string) func(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	QueryFunc := map[string]func(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error){
		"GetUser":         GetUser,
		"GetUserContract": GetUserContract,
		"GetBidders":      GetBidders,
		"ViewContracts":   ViewContracts,
		"GetUser":         GetUser,
		"GetUserBidds":    GetUserBidds,
		"GetContract":     GetContract,
		"GetVersion":      GetVersion,
	}
	return QueryFunction[fname]
}

////////////////////////////////////////////////////////////////
// SimpleChaincode - INVOKE Chaincode implementation
// User Can Invoke
// - Register a user using PostUser
// - Register an item using PostItem
// - The Owner of the item (User) can request that the item be put on auction
// - The Auction House can request that the auction request be Opened for bids using OpenAuctionForBids
// - One the auction is OPEN, registered buyers (Buyers) can send in bids vis PostBid
// - No bid is accepted when the status of the auction request is INIT or CLOSED
// - Either manually or by OpenAuctionRequest, the auction can be closed using CloseAuction
// - The CloseAuction creates a transaction and invokes PostTransaction
////////////////////////////////////////////////////////////////

func (t *SimpleChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	var err error
	var buff []byte

	//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	// Check Type of Transaction and apply business rules
	// before adding record to the block chain
	// In this version, the assumption is that args[1] specifies recType for all defined structs
	// Newer structs - the recType can be positioned anywhere and ChkReqType will check for recType
	// example:
	// ./peer chaincode invoke -l golang -n mycc -c '{"Function": "PostBid", "Args":["1111", "BID", "1", "1000", "300", "1200"]}'
	//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	if ChkReqType(args) == true {

		InvokeRequest := InvokeFunction(function)
		if InvokeRequest != nil {
			buff, err = InvokeRequest(stub, function, args)
		}
	} else {
		fmt.Println("Invoke() Invalid recType : ", args, "\n")
		return nil, errors.New("Invoke() : Invalid recType : " + args[0])
	}

	return buff, err
}

//////////////////////////////////////////////////////////////////////////////////////////
// SimpleChaincode - QUERY Chaincode implementation
// Client Can Query
// Sample Data
// ./peer chaincode query -l golang -n mycc -c '{"Function": "GetUser", "Args": ["4000"]}'
// ./peer chaincode query -l golang -n mycc -c '{"Function": "GetItem", "Args": ["2000"]}'
//////////////////////////////////////////////////////////////////////////////////////////

func (t *SimpleChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	var err error
	var buff []byte
	fmt.Println("ID Extracted and Type = ", args[0])
	fmt.Println("Args supplied : ", args)

	if len(args) < 1 {
		fmt.Println("Query() : Include at least 1 arguments Key ")
		return nil, errors.New("Query() : Expecting Transation type and Key value for query")
	}

	QueryRequest := QueryFunction(function)
	if QueryRequest != nil {
		buff, err = QueryRequest(stub, function, args)
	} else {
		fmt.Println("Query() Invalid function call : ", function)
		return nil, errors.New("Query() : Invalid function call : " + function)
	}

	if err != nil {
		fmt.Println("Query() Object not found : ", args[0])
		return nil, errors.New("Query() : Object not found : " + args[0])
	}
	return buff, err
}

func GetVersion(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	if len(args) < 1 {
		fmt.Println("GetVersion() : Requires 1 argument 'version'")
		return nil, errors.New("GetVersion() : Requires 1 argument 'version'")
	}
	// Get version from the ledger
	version, err := stub.Get
	State(args[0])
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for version\"}"
		return nil, errors.New(jsonResp)
	}

	if version == nil {
		jsonResp := "{\"Error\":\" service application version is invalid\"}"
		return nil, errors.New(jsonResp)
	}

	jsonResp := "{\"version\":\"" + string(version) + "\"}"
	fmt.Printf("Query Response:%s\n", jsonResp)
	return version, nil
}

//////////////////////////////////////////////////////////////////////////////////////////
// Retrieve User Information
// example:
// ./peer chaincode query -l golang -n mycc -c '{"Function": "GetUser", "Args": ["100"]}'
//
//////////////////////////////////////////////////////////////////////////////////////////
func GetUser(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	var err error

	// Get the Object and Display it
	Avalbytes, err := QueryLedger(stub, "UserTable", args)
	if err != nil {
		fmt.Println("GetUser() : Failed to Query Object ")
		jsonResp := "{\"Error\":\"Failed to get  Object Data for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	if Avalbytes == nil {
		fmt.Println("GetUser() : Incomplete Query Object ")
		jsonResp := "{\"Error\":\"Incomplete information about the key for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	fmt.Println("GetUser() : Response : Successfull -")
	return Avalbytes, nil
}

/////////////////////////////////////////////////////////////////////////////////////////
// Query callback representing the query of a chaincode
// Retrieve a Item by Item ID
// ./peer chaincode query -l golang -n mycc -c '{"Function": "GetContract", "Args": ["1000"]}'
/////////////////////////////////////////////////////////////////////////////////////////
func GetContract(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	var err error

	// Get the Objects and Display it
	Avalbytes, err := QueryLedger(stub, "ContractTable", args)
	if err != nil {
		fmt.Println("GetContract() : Failed to Query Object ")
		jsonResp := "{\"Error\":\"Failed to get  Object Data for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	if Avalbytes == nil {
		fmt.Println("GetContract() : Incomplete Query Object ")
		jsonResp := "{\"Error\":\"Incomplete information about the key for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	fmt.Println("GetContract() : Response : Successfull ")

	// Masking ItemImage binary data
	itemObj, _ := JSONtoAR(Avalbytes)
	itemObj.ItemImage = []byte{}
	Avalbytes, _ = ARtoJSON(itemObj)

	return Avalbytes, nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Retrieve a Bid based on two keys - AucID, BidNo
// A Bid has two Keys - The Auction Request Number and Bid Number
// ./peer chaincode query -l golang -n mycc -c '{"Function": "GetLastBid", "Args": ["1111"], "1"}'
//
///////////////////////////////////////////////////////////////////////////////////////////////////
func GetBid(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	var err error

	// Check there are 2 Arguments provided as per the the struct - two are computed
	// See example
	if len(args) < 2 {
		fmt.Println("GetBid(): Incorrect number of arguments. Expecting 2 ")
		fmt.Println("GetBid(): ./peer chaincode query -l golang -n mycc -c '{\"Function\": \"GetBid\", \"Args\": [\"1111\",\"6\"]}'")
		return nil, errors.New("GetBid(): Incorrect number of arguments. Expecting 2 ")
	}

	// Get the Objects and Display it
	Avalbytes, err := QueryLedger(stub, "BidTable", args)
	if err != nil {
		fmt.Println("GetBid() : Failed to Query Object ")
		jsonResp := "{\"Error\":\"Failed to get  Object Data for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	if Avalbytes == nil {
		fmt.Println("GetBid() : Incomplete Query Object ")
		jsonResp := "{\"Error\":\"Incomplete information about the key for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	fmt.Println("GetBid() : Response : Successfull -")
	return Avalbytes, nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////
// Retrieve Auction Closeout Information. When an Auction closes
// The highest bid is retrieved and converted to a Transaction
//  ./peer chaincode query -l golang -n mycc -c '{"Function": "GetTransaction", "Args": ["1111"]}'
//
///////////////////////////////////////////////////////////////////////////////////////////////////
func GetTransaction(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	//var err error

	// Get the Objects and Display it
	Avalbytes, err := QueryLedger(stub, "TransTable", args)
	if Avalbytes == nil {
		fmt.Println("GetTransaction() : Incomplete Query Object ")
		jsonResp := "{\"Error\":\"Incomplete information about the key for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	if err != nil {
		fmt.Println("GetTransaction() : Failed to Query Object ")
		jsonResp := "{\"Error\":\"Failed to get  Object Data for " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}

	fmt.Println("GetTransaction() : Response : Successfull")
	return Avalbytes, nil
}

/////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Create a master Object of the Item
// Since the Owner Changes hands, a record has to be written for each
// Transaction with the updated Encryption Key of the new owner
// Example
//./peer chaincode invoke -l golang -n mycc -c '{"Function": "PostItem", "Args":["1000", "ARTINV", "Shadows by Asppen", "Asppen Messer", "20140202", "Original", "Landscape" , "Canvas", "15 x 15 in", "sample_7.png","$600", "100"]}'
/////////////////////////////////////////////////////////////////////////////////////////////////////////////

func PostRequest(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	contractObject, err := CreateContract(args[0:])
	if err != nil {
		fmt.Println("PostRequest(): Cannot create item object \n")
		return nil, err
	}

	// Check if the Owner ID specified is registered and valid
	ownerInfo, err := ValidateMember(stub, ContractObject.UserID)
	fmt.Println("Owner information  ", ownerInfo, contractObject.UserID)
	if err != nil {
		fmt.Println("PostRequest() : Failed Owner information not found for ", contractObject.UserID)
		return nil, err
	}

	// Convert Item Object to JSON
	buff, err := ARtoJSON(contractObject) //
	if err != nil {
		fmt.Println("PostRequest() : Failed Cannot create object buffer for write : ", args[1])
		return nil, errors.New("PostRequest(): Failed Cannot create object buffer for write : " + args[1])
	} else {
		// Update the ledger with the Buffer Data
		// err = stub.PutState(args[0], buff)
		keys := []string{args[0]}
		err = UpdateLedger(stub, "ContractTable", keys, buff)
		if err != nil {
			fmt.Println("PostRequest() : write error while inserting record\n")
			return buff, err
		}

		// Put an entry into the Item History Table
		_, err = PostItemLog(stub, contractObject, "INITIAL", "DEFAULT")
		if err != nil {
			fmt.Println("PostRequestLog() : write error while inserting record\n")
			return nil, err
		}

		// Post Entry into ItemCatTable - i.e. Item Category Table
		// The first key 2016 is a dummy (band aid) key to extract all values
		keys = []string{"2016", args[4], args[0]}
		err = UpdateLedger(stub, "ContractCatTable", keys, buff)
		if err != nil {
			fmt.Println("PostRequest() : Write error while inserting record into ContractCatTable \n")
			return buff, err
		}

	}

	secret_key, _ := json.Marshal(contractObject.ContractId)
	fmt.Println(string(secret_key))
	return secret_key, nil
}

func CreateContract(args []string) (ContractObject, error) {

	var err error
	var myItem ContractObject

	// Check there are 11 Arguments provided as per the the struct - two are computed
	if len(args) != 11 {
		fmt.Println("CreateContract(): Incorrect number of arguments. Expecting 11 ")
		return myItem, errors.New("CreateContract(): Incorrect number of arguments. Expecting 11 ")
	}

	// Validate ItemID is an integer

	_, err = strconv.Atoi(args[0])
	if err != nil {
		fmt.Println("CreateContract(): ART ID should be an integer create failed! ")
		return myItem, errors.New("CreateContract(): contract ID should be an integer create failed!")
	}

	AES_key, _ := GenAESKey()

	// Append the AES Key, The Encrypted Image Byte Array and the file type
	myItem = ContractObject{AES_key, args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], "OPEN", args[10]}

	fmt.Println("CreateContract(): Item Object created: ID# ", myItem.ContractId, "\n AES Key: ", myItem.AES_Key)

	// Code to Validate the Item Object)
	// If User presents Crypto Key then key is used to validate the picture that is stored as part of the title
	// TODO

	return myItem, nil
}

///////////////////////////////////////////////////////////////////////////////////
// Since the Owner Changes hands, a record has to be written for each
// Transaction with the updated Encryption Key of the new owner
// This function is internally invoked by PostTransaction and is not a Public API
///////////////////////////////////////////////////////////////////////////////////

func UpdateItemObject(stub shim.ChaincodeStubInterface, ar []byte, hammerPrice string, buyer string) ([]byte, error) {

	var err error
	myItem, err := JSONtoAR(ar)
	if err != nil {
		fmt.Println("U() : UpdateItemObject() : Failed to create Art Record Object from JSON ")
		return nil, err
	}

	// Insert logic to  re-encrypt image by first fetching the current Key
	CurrentAES_Key := myItem.AES_Key
	// Decrypt Image and Save Image in a file
	image := Decrypt(CurrentAES_Key, myItem.ItemImage)

	// Get a New Key & Encrypt Image with New Key
	myItem.AES_Key, _ = GenAESKey()
	myItem.ItemImage = Encrypt(myItem.AES_Key, image)

	// Update the owner to the Buyer and update price to auction hammer price
	myItem.ItemBasePrice = hammerPrice
	myItem.CurrentOwnerID = buyer

	ar, err = ARtoJSON(myItem)
	keys := []string{myItem.ItemID, myItem.CurrentOwnerID}
	err = ReplaceLedgerEntry(stub, "ItemTable", keys, ar)
	if err != nil {
		fmt.Println("UpdateItemObject() : Failed ReplaceLedgerEntry in ItemTable into Blockchain ")
		return nil, err
	}
	fmt.Println("UpdateItemObject() : ReplaceLedgerEntry in ItemTable successful ")

	// Update entry in Item Category Table as it holds the Item object as wekk
	keys = []string{"2016", myItem.ItemSubject, myItem.ItemID}
	err = ReplaceLedgerEntry(stub, "ItemCatTable", keys, ar)
	if err != nil {
		fmt.Println("UpdateItemObject() : Failed ReplaceLedgerEntry in ItemCategoryTable into Blockchain ")
		return nil, err
	}

	fmt.Println("UpdateItemObject() : ReplaceLedgerEntry in ItemCategoryTable successful ")
	return myItem.AES_Key, nil
}

//////////////////////////////////////////////////////////
// Create an Item Transaction record to process Request
// This is invoked by the CloseAuctionRequest
//
////////////////////////////////////////////////////////////
func PostTransaction(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	if function != "PostTransaction" {
		return nil, errors.New("PostTransaction(): Invalid function name. Expecting \"PostTransaction\"")
	}

	ar, err := CreateTransactionRequest(args[0:]) //
	if err != nil {
		return nil, err
	}

	// Validate buyer's ID
	buyer, err := ValidateMember(stub, ar.UserId)
	if err != nil {
		fmt.Println("PostTransaction() : Failed Buyer not Registered in Blockchain ", ar.UserId)
		return nil, err
	}

	fmt.Println("PostTransaction(): Validated Buyer information successfully ", buyer, ar.UserId)

	// Validate Item record
	lastUpdatedItemOBCObject, err := ValidateItemSubmission(stub, ar.ItemID)
	if err != nil {
		fmt.Println("PostTransaction() : Failed Could not Validate Item Object in Blockchain ", ar.ItemID)
		return lastUpdatedItemOBCObject, err
	}
	fmt.Println("PostTransaction() : Validated Item Object in Blockchain successfully", ar.ItemID)

	// Update Item Object with new Owner Key
	newKey, err := UpdateItemObject(stub, lastUpdatedItemOBCObject, ar.HammerPrice, ar.UserId)
	if err != nil {
		fmt.Println("PostTransaction() : Failed to update Item Master Object in Blockchain ", ar.ItemID)
		return nil, err
	} else {
		// Write New Key to file
		fmt.Println("PostTransaction() : New encryption Key is  ", newKey)
	}
	fmt.Println("PostTransaction() : Updated Item Master Object in Blockchain successfully", ar.ItemID)

	// Post an Item Log
	itemObject, err := JSONtoAR(lastUpdatedItemOBCObject)
	if err != nil {
		fmt.Println("PostTransaction() : Conversion error JSON to ItemRecord\n")
		return lastUpdatedItemOBCObject, err
	}

	// A life cycle event is added to say that the Item is no longer on auction
	itemObject.ItemBasePrice = ar.HammerPrice
	itemObject.CurrentOwnerID = ar.UserId

	_, err = PostItemLog(stub, itemObject, "NA", "DEFAULT")
	if err != nil {
		fmt.Println("PostTransaction() : write error while inserting item log record\n")
		return lastUpdatedItemOBCObject, err
	}

	fmt.Println("PostTransaction() : Inserted item log record successfully", ar.ItemID)

	// Convert Transaction Object to JSON
	buff, err := TrantoJSON(ar) //
	if err != nil {
		fmt.Println("GetObjectBuffer() : Failed to convert Transaction Object to JSON ", args[0])
		return nil, err
	}

	// Update the ledger with the Buffer Data
	keys := []string{args[0], args[3]}
	err = UpdateLedger(stub, "TransTable", keys, buff)
	if err != nil {
		fmt.Println("PostTransaction() : write error while inserting record\n")
		return buff, err
	}

	fmt.Println("PostTransaction() : Posted Transaction Record successfully\n")

	// Returns New Key. To get Transaction Details, run GetTransaction

	secret_key, _ := json.Marshal(newKey)
	fmt.Println(string(secret_key))
	return secret_key, nil

}

func CreateTransactionRequest(args []string) (ItemTransaction, error) {

	var at ItemTransaction

	// Check there are 9 Arguments
	if len(args) != 9 {
		fmt.Println("CreateTransactionRequest(): Incorrect number of arguments. Expecting 9 ")
		return at, errors.New("CreateTransactionRequest() : Incorrect number of arguments. Expecting 9 ")
	}

	at = ItemTransaction{args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8]}
	fmt.Println("CreateTransactionRequest() : Transaction Request: ", at)

	return at, nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Create a Bid Object
// Once an Item has been opened for auction, bids can be submitted as long as the auction is "OPEN"
//./peer chaincode invoke -l golang -n mycc -c '{"Function": "PostBid", "Args":["1111", "BID", "1", "1000", "300", "1200"]}'
//./peer chaincode invoke -l golang -n mycc -c '{"Function": "PostBid", "Args":["1111", "BID", "2", "1000", "400", "3000"]}'
//
/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func PostBid(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	bid, err := CreateBidObject(args[0:]) //
	if err != nil {
		return nil, err
	}

	// Reject the Bid if the Buyer Information Is not Valid or not registered on the Block Chain
	buyerInfo, err := ValidateMember(stub, args[4])
	fmt.Println("Buyer information  ", buyerInfo, "  ", args[4])
	if err != nil {
		fmt.Println("PostBid() : Failed Buyer not registered on the block-chain ", args[4])
		return nil, err
	}

	///////////////////////////////////////
	// Reject Bid if Auction is not "OPEN"
	///////////////////////////////////////
	RBytes, err := GetAuctionRequest(stub, "GetAuctionRequest", []string{args[0]})
	if err != nil {
		fmt.Println("PostBid() : Cannot find Auction record ", args[0])
		return nil, errors.New("PostBid(): Cannot find Auction record : " + args[0])
	}

	aucR, err := JSONtoAucReq(RBytes)
	if err != nil {
		fmt.Println("PostBid() : Cannot UnMarshall Auction record")
		return nil, errors.New("PostBid(): Cannot UnMarshall Auction record: " + args[0])
	}

	if aucR.Status != "OPEN" {
		fmt.Println("PostBid() : Cannot accept Bid as Auction is not OPEN ", args[0])
		return nil, errors.New("PostBid(): Cannot accept Bid as Auction is not OPEN : " + args[0])
	}

	///////////////////////////////////////////////////////////////////
	// Reject Bid if the time bid was received is > Auction Close Time
	///////////////////////////////////////////////////////////////////
	if tCompare(bid.BidTime, aucR.CloseDate) == false {
		fmt.Println("PostBid() Failed : BidTime past the Auction Close Time")
		return nil, fmt.Errorf("PostBid() Failed : BidTime past the Auction Close Time %s, %s", bid.BidTime, aucR.CloseDate)
	}

	//////////////////////////////////////////////////////////////////
	// Reject Bid if Item ID on Bid does not match Item ID on Auction
	//////////////////////////////////////////////////////////////////
	if aucR.ItemID != bid.ItemID {
		fmt.Println("PostBid() Failed : Item ID mismatch on bid. Bid Rejected")
		return nil, errors.New("PostBid() : Item ID mismatch on Bid. Bid Rejected")
	}

	//////////////////////////////////////////////////////////////////////
	// Reject Bid if Bid Price is less than Reserve Price
	// Convert Bid Price and Reserve Price to Integer (TODO - Float)
	//////////////////////////////////////////////////////////////////////
	bp, err := strconv.Atoi(bid.BidPrice)
	if err != nil {
		fmt.Println("PostBid() Failed : Bid price should be an integer")
		return nil, errors.New("PostBid() : Bid price should be an integer")
	}

	hp, err := strconv.Atoi(aucR.ReservePrice)
	if err != nil {
		return nil, errors.New("PostItem() : Reserve Price should be an integer")
	}

	// Check if Bid Price is > Auction Request Reserve Price
	if bp < hp {
		return nil, errors.New("PostItem() : Bid Price must be greater than Reserve Price")
	}

	////////////////////////////
	// Post or Accept the Bid
	////////////////////////////
	buff, err := BidtoJSON(bid) //

	if err != nil {
		fmt.Println("PostBid() : Failed Cannot create object buffer for write : ", args[1])
		return nil, errors.New("PostBid(): Failed Cannot create object buffer for write : " + args[1])
	} else {
		// Update the ledger with the Buffer Data
		// err = stub.PutState(args[0], buff)
		keys := []string{args[0], args[2]}
		err = UpdateLedger(stub, "BidTable", keys, buff)
		if err != nil {
			fmt.Println("PostBidTable() : write error while inserting record\n")
			return buff, err
		}
	}

	return buff, err
}

func CreateBidObject(args []string) (Bid, error) {
	var err error
	var aBid Bid

	// Check there are 11 Arguments
	// See example
	if len(args) != 6 {
		fmt.Println("CreateBidObject(): Incorrect number of arguments. Expecting 6 ")
		return aBid, errors.New("CreateBidObject() : Incorrect number of arguments. Expecting 6 ")
	}

	// Validate Bid is an integer

	_, err = strconv.Atoi(args[0])
	if err != nil {
		return aBid, errors.New("CreateBidObject() : Bid ID should be an integer")
	}

	_, err = strconv.Atoi(args[2])
	if err != nil {
		return aBid, errors.New("CreateBidObject() : Bid ID should be an integer")
	}

	bidTime := time.Now().Format("2006-01-02 15:04:05")

	aBid = Bid{args[0], args[1], args[2], args[3], args[4], args[5], bidTime}
	fmt.Println("CreateBidObject() : Bid Object : ", aBid)

	return aBid, nil
}

///////////////////////////////////////////////////////////////////////
// Encryption and Decryption Section
// Images will be Encrypted and stored and the key will be part of the
// certificate that is provided to the Owner
///////////////////////////////////////////////////////////////////////

const (
	AESKeyLength = 32 // AESKeyLength is the default AES key length
	NonceSize    = 24 // NonceSize is the default NonceSize
)

///////////////////////////////////////////////////
// GetRandomBytes returns len random looking bytes
///////////////////////////////////////////////////
func GetRandomBytes(len int) ([]byte, error) {
	key := make([]byte, len)

	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}

	return key, nil
}

////////////////////////////////////////////////////////////
// GenAESKey returns a random AES key of length AESKeyLength
// 3 Functions to support Encryption and Decryption
// GENAESKey() - Generates AES symmetric key
// Encrypt() Encrypts a [] byte
// Decrypt() Decryts a [] byte
////////////////////////////////////////////////////////////
func GenAESKey() ([]byte, error) {
	return GetRandomBytes(AESKeyLength)
}

func PKCS5Pad(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	pad := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, pad...)
}

func PKCS5Unpad(src []byte) []byte {
	len := len(src)
	unpad := int(src[len-1])
	return src[:(len - unpad)]
}

func Decrypt(key []byte, ciphertext []byte) []byte {

	// Create the AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// Before even testing the decryption,
	// if the text is too small, then it is incorrect
	if len(ciphertext) < aes.BlockSize {
		panic("Text is too short")
	}

	// Get the 16 byte IV
	iv := ciphertext[:aes.BlockSize]

	// Remove the IV from the ciphertext
	ciphertext = ciphertext[aes.BlockSize:]

	// Return a decrypted stream
	stream := cipher.NewCFBDecrypter(block, iv)

	// Decrypt bytes from ciphertext
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext
}

func Encrypt(key []byte, ba []byte) []byte {

	// Create the AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	// Empty array of 16 + ba length
	// Include the IV at the beginning
	ciphertext := make([]byte, aes.BlockSize+len(ba))

	// Slice of first 16 bytes
	iv := ciphertext[:aes.BlockSize]

	// Write 16 rand bytes to fill iv
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	// Return an encrypted stream
	stream := cipher.NewCFBEncrypter(block, iv)

	// Encrypt bytes from ba to ciphertext
	stream.XORKeyStream(ciphertext[aes.BlockSize:], ba)

	return ciphertext
}

//////////////////////////////////////////////////////////
// JSON To args[] - return a map of the JSON string
//////////////////////////////////////////////////////////
func JSONtoArgs(Avalbytes []byte) (map[string]interface{}, error) {

	var data map[string]interface{}

	if err := json.Unmarshal(Avalbytes, &data); err != nil {
		return nil, err
	}

	return data, nil
}

//////////////////////////////////////////////////////////
// Variation of the above - return value from a JSON string
//////////////////////////////////////////////////////////

func GetKeyValue(Avalbytes []byte, key string) string {
	var dat map[string]interface{}
	if err := json.Unmarshal(Avalbytes, &dat); err != nil {
		panic(err)
	}

	val := dat[key].(string)
	return val
}

//////////////////////////////////////////////////////////
// Time and Date Comparison
// tCompare("2016-06-28 18:40:57", "2016-06-27 18:45:39")
//////////////////////////////////////////////////////////
func tCompare(t1 string, t2 string) bool {

	layout := "2006-01-02 15:04:05"
	bidTime, err := time.Parse(layout, t1)
	if err != nil {
		fmt.Println("tCompare() Failed : time Conversion error on t1")
		return false
	}

	aucCloseTime, err := time.Parse(layout, t2)
	if err != nil {
		fmt.Println("tCompare() Failed : time Conversion error on t2")
		return false
	}

	if bidTime.Before(aucCloseTime) {
		return true
	}

	return false
}

//////////////////////////////////////////////////////////
// Converts JSON String to an ART Object
//////////////////////////////////////////////////////////
func JSONtoAR(data []byte) (ContractObject, error) {

	ar := ItemObject{}
	err := json.Unmarshal([]byte(data), &ar)
	if err != nil {
		fmt.Println("Unmarshal failed : ", err)
	}

	return ar, err
}

//////////////////////////////////////////////////////////
// Converts an ART Object to a JSON String
//////////////////////////////////////////////////////////
func ARtoJSON(ar ContractObject) ([]byte, error) {

	ajson, err := json.Marshal(ar)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return ajson, nil
}

//////////////////////////////////////////////////////////
// Converts an BID to a JSON String
//////////////////////////////////////////////////////////
func ItemLogtoJSON(item ContractObject) ([]byte, error) {

	ajson, err := json.Marshal(item)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return ajson, nil
}

//////////////////////////////////////////////////////////
// Converts an User Object to a JSON String
//////////////////////////////////////////////////////////
func JSONtoItemLog(ithis []byte) (ContractObject, error) {

	item := ContractObject{}
	err := json.Unmarshal(ithis, &item)
	if err != nil {
		fmt.Println("JSONtoContractObject error: ", err)
		return item, err
	}
	return item, err
}

//////////////////////////////////////////////////////////
// Converts an Auction Request to a JSON String
//////////////////////////////////////////////////////////
func AucReqtoJSON(ar ContractObject) ([]byte, error) {

	ajson, err := json.Marshal(ar)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return ajson, nil
}

//////////////////////////////////////////////////////////
// Converts an User Object to a JSON String
//////////////////////////////////////////////////////////
func JSONtoAucReq(areq []byte) (ContractObject, error) {

	ar := ContractObject{}
	err := json.Unmarshal(areq, &ar)
	if err != nil {
		fmt.Println("JSONtoAucReq error: ", err)
		return ar, err
	}
	return ar, err
}

//////////////////////////////////////////////////////////
// Converts BID Object to JSON String
//////////////////////////////////////////////////////////
func BidtoJSON(myHand Bid) ([]byte, error) {

	ajson, err := json.Marshal(myHand)
	if err != nil {
		fmt.Println("BidtoJSON error: ", err)
		return nil, err
	}
	return ajson, nil
}

//////////////////////////////////////////////////////////
// Converts JSON String to BID Object
//////////////////////////////////////////////////////////
func JSONtoBid(areq []byte) (Bid, error) {

	myHand := Bid{}
	err := json.Unmarshal(areq, &myHand)
	if err != nil {
		fmt.Println("JSONtoBid error: ", err)
		return myHand, err
	}
	return myHand, err
}

//////////////////////////////////////////////////////////
// Converts an User Object to a JSON String
//////////////////////////////////////////////////////////
func UsertoJSON(user UserObject) ([]byte, error) {

	ajson, err := json.Marshal(user)
	if err != nil {
		fmt.Println("UsertoJSON error: ", err)
		return nil, err
	}
	fmt.Println("UsertoJSON created: ", ajson)
	return ajson, nil
}

//////////////////////////////////////////////////////////
// Converts an User Object to a JSON String
//////////////////////////////////////////////////////////
func JSONtoUser(user []byte) (UserObject, error) {

	ur := UserObject{}
	err := json.Unmarshal(user, &ur)
	if err != nil {
		fmt.Println("JSONtoUser error: ", err)
		return ur, err
	}
	fmt.Println("JSONtoUser created: ", ur)
	return ur, err
}

//////////////////////////////////////////////////////////
// Converts an Item Transaction to a JSON String
//////////////////////////////////////////////////////////
func TrantoJSON(at ItemTransaction) ([]byte, error) {

	ajson, err := json.Marshal(at)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return ajson, nil
}

//////////////////////////////////////////////////////////
// Converts an Trans Object to a JSON String
//////////////////////////////////////////////////////////
func JSONtoTran(areq []byte) (ItemTransaction, error) {

	at := ItemTransaction{}
	err := json.Unmarshal(areq, &at)
	if err != nil {
		fmt.Println("JSONtoTran error: ", err)
		return at, err
	}
	return at, err
}

//////////////////////////////////////////////
// Validates an ID for Well Formed
//////////////////////////////////////////////

func validateID(id string) error {
	// Validate UserID is an integer

	_, err := strconv.Atoi(id)
	if err != nil {
		return errors.New("validateID(): User ID should be an integer")
	}
	return nil
}

//////////////////////////////////////////////
// Create an ItemLog from Item
//////////////////////////////////////////////

func ItemToItemLog(io ItemObject) ItemLog {

	iLog := ItemLog{}
	iLog.ItemID = io.ItemID
	iLog.Status = "INITIAL"
	iLog.AuctionedBy = "DEFAULT"
	iLog.RecType = "ILOG"
	iLog.ItemDesc = io.ItemDesc
	iLog.CurrentOwner = io.CurrentOwnerID
	iLog.Date = time.Now().Format("2006-01-02 15:04:05")

	return iLog
}

//////////////////////////////////////////////
// Convert Bid to Transaction for Posting
//////////////////////////////////////////////

func BidtoTransaction(bid Bid) ItemTransaction {

	var t ItemTransaction
	t.AuctionID = bid.AuctionID
	t.RecType = "POSTTRAN"
	t.ItemID = bid.ItemID
	t.TransType = "SALE"
	t.UserId = bid.BuyerID
	t.TransDate = time.Now().Format("2006-01-02 15:04:05")
	t.HammerTime = bid.BidTime
	t.HammerPrice = bid.BidPrice
	t.Details = "The Highest Bidder does not always win"

	return t
}

////////////////////////////////////////////////////////////////////////////
// Validate if the User Information Exists
// in the block-chain
////////////////////////////////////////////////////////////////////////////
func ValidateMember(stub shim.ChaincodeStubInterface, owner string) ([]byte, error) {

	// Get the Item Objects and Display it
	// Avalbytes, err := stub.GetState(owner)
	args := []string{owner, "USER"}
	Avalbytes, err := QueryLedger(stub, "UserTable", args)

	if err != nil {
		fmt.Println("ValidateMember() : Failed - Cannot find valid owner record for ART  ", owner)
		jsonResp := "{\"Error\":\"Failed to get Owner Object Data for " + owner + "\"}"
		return nil, errors.New(jsonResp)
	}

	if Avalbytes == nil {
		fmt.Println("ValidateMember() : Failed - Incomplete owner record for ART  ", owner)
		jsonResp := "{\"Error\":\"Failed - Incomplete information about the owner for " + owner + "\"}"
		return nil, errors.New(jsonResp)
	}

	fmt.Println("ValidateMember() : Validated Item Owner:\n", owner)
	return Avalbytes, nil
}

////////////////////////////////////////////////////////////////////////////
// Open a Ledgers if one does not exist
// These ledgers will be used to write /  read data
// Use names are listed in aucTables {}
// THIS FUNCTION REPLACES ALL THE INIT Functions below
//  - InitUserReg()
//  - InitAucReg()
//  - InitBidReg()
//  - InitItemReg()
//  - InitItemMaster()
//  - InitTransReg()
//  - InitAuctionTriggerReg()
//  - etc. etc.
////////////////////////////////////////////////////////////////////////////
func InitLedger(stub shim.ChaincodeStubInterface, tableName string) error {

	// Generic Table Creation Function - requires Table Name and Table Key Entry
	// Create Table - Get number of Keys the tables supports
	// This version assumes all Keys are String and the Data is Bytes
	// This Function can replace all other InitLedger function in this app such as InitItemLedger()

	nKeys := GetNumberOfKeys(tableName)
	if nKeys < 1 {
		fmt.Println("Atleast 1 Key must be provided \n")
		fmt.Println("Auction_Application: Failed creating Table ", tableName)
		return errors.New("Auction_Application: Failed creating Table " + tableName)
	}

	var columnDefsForTbl []*shim.ColumnDefinition

	for i := 0; i < nKeys; i++ {
		columnDef := shim.ColumnDefinition{Name: "keyName" + strconv.Itoa(i), Type: shim.ColumnDefinition_STRING, Key: true}
		columnDefsForTbl = append(columnDefsForTbl, &columnDef)
	}

	columnLastTblDef := shim.ColumnDefinition{Name: "Details", Type: shim.ColumnDefinition_BYTES, Key: false}
	columnDefsForTbl = append(columnDefsForTbl, &columnLastTblDef)

	// Create the Table (Nil is returned if the Table exists or if the table is created successfully
	err := stub.CreateTable(tableName, columnDefsForTbl)

	if err != nil {
		fmt.Println("Auction_Application: Failed creating Table ", tableName)
		return errors.New("Auction_Application: Failed creating Table " + tableName)
	}

	return err
}

////////////////////////////////////////////////////////////////////////////
// Open a User Registration Table if one does not exist
// Register users into this table
////////////////////////////////////////////////////////////////////////////
func UpdateLedger(stub shim.ChaincodeStubInterface, tableName string, keys []string, args []byte) error {

	nKeys := GetNumberOfKeys(tableName)
	if nKeys < 1 {
		fmt.Println("Atleast 1 Key must be provided \n")
	}

	var columns []*shim.Column

	for i := 0; i < nKeys; i++ {
		col := shim.Column{Value: &shim.Column_String_{String_: keys[i]}}
		columns = append(columns, &col)
	}

	lastCol := shim.Column{Value: &shim.Column_Bytes{Bytes: []byte(args)}}
	columns = append(columns, &lastCol)

	row := shim.Row{columns}
	ok, err := stub.InsertRow(tableName, row)
	if err != nil {
		return fmt.Errorf("UpdateLedger: InsertRow into "+tableName+" Table operation failed. %s", err)
	}
	if !ok {
		return errors.New("UpdateLedger: InsertRow into " + tableName + " Table failed. Row with given key " + keys[0] + " already exists")
	}

	fmt.Println("UpdateLedger: InsertRow into ", tableName, " Table operation Successful. ")
	return nil
}

////////////////////////////////////////////////////////////////////////////
// Open a User Registration Table if one does not exist
// Register users into this table
////////////////////////////////////////////////////////////////////////////
func DeleteFromLedger(stub shim.ChaincodeStubInterface, tableName string, keys []string) error {
	var columns []shim.Column

	//nKeys := GetNumberOfKeys(tableName)
	nCol := len(keys)
	if nCol < 1 {
		fmt.Println("Atleast 1 Key must be provided \n")
		return errors.New("DeleteFromLedger failed. Must include at least key values")
	}

	for i := 0; i < nCol; i++ {
		colNext := shim.Column{Value: &shim.Column_String_{String_: keys[i]}}
		columns = append(columns, colNext)
	}

	err := stub.DeleteRow(tableName, columns)
	if err != nil {
		return fmt.Errorf("DeleteFromLedger operation failed. %s", err)
	}

	fmt.Println("DeleteFromLedger: DeleteRow from ", tableName, " Table operation Successful. ")
	return nil
}

////////////////////////////////////////////////////////////////////////////
// Replaces the Entry in the Ledger
//
////////////////////////////////////////////////////////////////////////////
func ReplaceLedgerEntry(stub shim.ChaincodeStubInterface, tableName string, keys []string, args []byte) error {

	nKeys := GetNumberOfKeys(tableName)
	if nKeys < 1 {
		fmt.Println("Atleast 1 Key must be provided \n")
	}

	var columns []*shim.Column

	for i := 0; i < nKeys; i++ {
		col := shim.Column{Value: &shim.Column_String_{String_: keys[i]}}
		columns = append(columns, &col)
	}

	lastCol := shim.Column{Value: &shim.Column_Bytes{Bytes: []byte(args)}}
	columns = append(columns, &lastCol)

	row := shim.Row{columns}
	ok, err := stub.ReplaceRow(tableName, row)
	if err != nil {
		return fmt.Errorf("ReplaceLedgerEntry: Replace Row into "+tableName+" Table operation failed. %s", err)
	}
	if !ok {
		return errors.New("ReplaceLedgerEntry: Replace Row into " + tableName + " Table failed. Row with given key " + keys[0] + " already exists")
	}

	fmt.Println("ReplaceLedgerEntry: Replace Row in ", tableName, " Table operation Successful. ")
	return nil
}

////////////////////////////////////////////////////////////////////////////
// Query a User Object by Table Name and Key
////////////////////////////////////////////////////////////////////////////
func QueryLedger(stub shim.ChaincodeStubInterface, tableName string, args []string) ([]byte, error) {

	var columns []shim.Column
	nCol := GetNumberOfKeys(tableName)
	for i := 0; i < nCol; i++ {
		colNext := shim.Column{Value: &shim.Column_String_{String_: args[i]}}
		columns = append(columns, colNext)
	}

	row, err := stub.GetRow(tableName, columns)
	fmt.Println("Length or number of rows retrieved ", len(row.Columns))

	if len(row.Columns) == 0 {
		jsonResp := "{\"Error\":\"Failed retrieving data " + args[0] + ". \"}"
		fmt.Println("Error retrieving data record for Key = ", args[0], "Error : ", jsonResp)
		return nil, errors.New(jsonResp)
	}

	//fmt.Println("User Query Response:", row)
	//jsonResp := "{\"Owner\":\"" + string(row.Columns[nCol].GetBytes()) + "\"}"
	//fmt.Println("User Query Response:%s\n", jsonResp)
	Avalbytes := row.Columns[nCol].GetBytes()

	// Perform Any additional processing of data
	fmt.Println("QueryLedger() : Successful - Proceeding to ProcessRequestType ")
	err = ProcessQueryResult(stub, Avalbytes, args)
	if err != nil {
		fmt.Println("QueryLedger() : Cannot create object  : ", args[1])
		jsonResp := "{\"QueryLedger() Error\":\" Cannot create Object for key " + args[0] + "\"}"
		return nil, errors.New(jsonResp)
	}
	return Avalbytes, nil
}

/////////////////////////////////////////////////////////////////////////////////////////////////////
// Get List of Bids for an Auction
// in the block-chain --
// ./peer chaincode query -l golang -n mycc -c '{"Function": "GetListOfBids", "Args": ["1111"]}'
// ./peer chaincode query -l golang -n mycc -c '{"Function": "GetLastBid", "Args": ["1111"]}'
// ./peer chaincode query -l golang -n mycc -c '{"Function": "GetHighestBid", "Args": ["1111"]}'
/////////////////////////////////////////////////////////////////////////////////////////////////////
func GetListOfBids(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	rows, err := GetList(stub, "BidTable", args)
	if err != nil {
		return nil, fmt.Errorf("GetListOfBids operation failed. Error marshaling JSON: %s", err)
	}

	nCol := GetNumberOfKeys("BidTable")

	tlist := make([]Bid, len(rows))
	for i := 0; i < len(rows); i++ {
		ts := rows[i].Columns[nCol].GetBytes()
		bid, err := JSONtoBid(ts)
		if err != nil {
			fmt.Println("GetListOfBids() Failed : Ummarshall error")
			return nil, fmt.Errorf("GetListOfBids() operation failed. %s", err)
		}
		tlist[i] = bid
	}

	jsonRows, _ := json.Marshal(tlist)

	fmt.Println("List of Bids Requested : ", jsonRows)
	return jsonRows, nil

}

////////////////////////////////////////////////////////////////////////////
// Get List of Open Auctions  for which bids can be supplied
// in the block-chain
// This is a fixed Query to be issued as below
// ./peer chaincode query -l golang -n mycc -c '{"Function": "GetListOfOpenAucs", "Args": ["2016"]}'
////////////////////////////////////////////////////////////////////////////
func GetListOfOpenContracts(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	rows, err := GetList(stub, "ContractOpenTable", args)
	if err != nil {
		return nil, fmt.Errorf("GetListOfOpenContracts operation failed. Error marshaling JSON: %s", err)
	}

	nCol := GetNumberOfKeys("ContractOpenTable")

	tlist := make([]AuctionRequest, len(rows))
	for i := 0; i < len(rows); i++ {
		ts := rows[i].Columns[nCol].GetBytes()
		ar, err := JSONtoAucReq(ts)
		if err != nil {
			fmt.Println("GetListOfOpenAucs() Failed : Ummarshall error")
			return nil, fmt.Errorf("GetListOfOpenAucs() operation failed. %s", err)
		}
		tlist[i] = ar
	}

	jsonRows, _ := json.Marshal(tlist)

	//fmt.Println("List of Open Auctions : ", jsonRows)
	return jsonRows, nil

}

////////////////////////////////////////////////////////////////////////////
// Get the Item History for an Item
// in the block-chain .. Pass the Item ID
// ./peer chaincode query -l golang -n mycc -c '{"Function": "GetItemLog", "Args": ["1000"]}'
////////////////////////////////////////////////////////////////////////////
func GetItemLog(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	// Check there are 1 Arguments provided as per the the struct - two are computed
	// See example
	if len(args) < 1 {
		fmt.Println("GetItemLog(): Incorrect number of arguments. Expecting 1 ")
		fmt.Println("GetItemLog(): ./peer chaincode query -l golang -n mycc -c '{\"Function\": \"GetItem\", \"Args\": [\"1111\"]}'")
		return nil, errors.New("CreateItemObject(): Incorrect number of arguments. Expecting 12 ")
	}

	rows, err := GetList(stub, "ItemHistoryTable", args)
	if err != nil {
		return nil, fmt.Errorf("GetItemLog() operation failed. Error marshaling JSON: %s", err)
	}
	nCol := GetNumberOfKeys("ItemHistoryTable")

	tlist := make([]ItemLog, len(rows))
	for i := 0; i < len(rows); i++ {
		ts := rows[i].Columns[nCol].GetBytes()
		il, err := JSONtoItemLog(ts)
		if err != nil {
			fmt.Println("() Failed : Ummarshall error")
			return nil, fmt.Errorf("GetItemLog() operation failed. %s", err)
		}
		tlist[i] = il
	}

	jsonRows, _ := json.Marshal(tlist)

	//fmt.Println("All History : ", jsonRows)
	return jsonRows, nil

}

////////////////////////////////////////////////////////////////////////////
// Get a List of Items by Category
// in the block-chain
// Input is 2016 + Category
// Categories include whatever has been defined in the Item Tables - Landscape, Modern, ...
// See Sample data
// ./peer chaincode query -l golang -n mycc -c '{"Function": "GetItemListByCat", "Args": ["2016", "Modern"]}'
////////////////////////////////////////////////////////////////////////////
func GetItemListByCat(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	// Check there are 1 Arguments provided as per the the struct - two are computed
	// See example
	if len(args) < 1 {
		fmt.Println("GetItemListByCat(): Incorrect number of arguments. Expecting 1 ")
		fmt.Println("GetItemListByCat(): ./peer chaincode query -l golang -n mycc -c '{\"Function\": \"GetItemListByCat\", \"Args\": [\"Modern\"]}'")
		return nil, errors.New("CreateItemObject(): Incorrect number of arguments. Expecting 1 ")
	}

	rows, err := GetList(stub, "ItemCatTable", args)
	if err != nil {
		return nil, fmt.Errorf("GetItemListByCat() operation failed. Error GetList: %s", err)
	}

	nCol := GetNumberOfKeys("ItemCatTable")

	tlist := make([]ItemObject, len(rows))
	for i := 0; i < len(rows); i++ {
		ts := rows[i].Columns[nCol].GetBytes()
		io, err := JSONtoAR(ts)
		if err != nil {
			fmt.Println("() Failed : Ummarshall error")
			return nil, fmt.Errorf("GetItemListByCat() operation failed. %s", err)
		}
		//TODO: Masking Image binary data, Need a clean solution ?
		io.ItemImage = []byte{}
		tlist[i] = io
	}

	jsonRows, _ := json.Marshal(tlist)

	//fmt.Println("All Items : ", jsonRows)
	return jsonRows, nil

}

////////////////////////////////////////////////////////////////////////////
// Get a List of Users by Category
// in the block-chain
////////////////////////////////////////////////////////////////////////////
func GetUserListByCat(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	// Check there are 1 Arguments provided as per the the struct - two are computed
	// See example
	if len(args) < 1 {
		fmt.Println("GetUserListByCat(): Incorrect number of arguments. Expecting 1 ")
		fmt.Println("GetUserListByCat(): ./peer chaincode query -l golang -n mycc -c '{\"Function\": \"GetUserListByCat\", \"Args\": [\"AH\"]}'")
		return nil, errors.New("CreateUserObject(): Incorrect number of arguments. Expecting 1 ")
	}

	rows, err := GetList(stub, "UserCatTable", args)
	if err != nil {
		return nil, fmt.Errorf("GetUserListByCat() operation failed. Error marshaling JSON: %s", err)
	}

	nCol := GetNumberOfKeys("UserCatTable")

	tlist := make([]UserObject, len(rows))
	for i := 0; i < len(rows); i++ {
		ts := rows[i].Columns[nCol].GetBytes()
		uo, err := JSONtoUser(ts)
		if err != nil {
			fmt.Println("GetUserListByCat() Failed : Ummarshall error")
			return nil, fmt.Errorf("GetUserListByCat() operation failed. %s", err)
		}
		tlist[i] = uo
	}

	jsonRows, _ := json.Marshal(tlist)

	//fmt.Println("All Users : ", jsonRows)
	return jsonRows, nil

}

////////////////////////////////////////////////////////////////////////////
// Get a List of Rows based on query criteria from the OBC
//
////////////////////////////////////////////////////////////////////////////
func GetList(stub shim.ChaincodeStubInterface, tableName string, args []string) ([]shim.Row, error) {
	var columns []shim.Column

	nKeys := GetNumberOfKeys(tableName)
	nCol := len(args)
	if nCol < 1 {
		fmt.Println("Atleast 1 Key must be provided \n")
		return nil, errors.New("GetList failed. Must include at least key values")
	}

	for i := 0; i < nCol; i++ {
		colNext := shim.Column{Value: &shim.Column_String_{String_: args[i]}}
		columns = append(columns, colNext)
	}

	rowChannel, err := stub.GetRows(tableName, columns)
	if err != nil {
		return nil, fmt.Errorf("GetList operation failed. %s", err)
	}
	var rows []shim.Row
	for {
		select {
		case row, ok := <-rowChannel:
			if !ok {
				rowChannel = nil
			} else {
				rows = append(rows, row)
				//If required enable for debugging
				//fmt.Println(row)
			}
		}
		if rowChannel == nil {
			break
		}
	}

	fmt.Println("Number of Keys retrieved : ", nKeys)
	fmt.Println("Number of rows retrieved : ", len(rows))
	return rows, nil
}

////////////////////////////////////////////////////////////////////////////
// Get The Highest Bid Received so far for an Auction
// in the block-chain
////////////////////////////////////////////////////////////////////////////
func GetLastBid(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	tn := "BidTable"
	rows, err := GetList(stub, tn, args)
	if err != nil {
		return nil, fmt.Errorf("GetLastBid operation failed. %s", err)
	}
	nCol := GetNumberOfKeys(tn)
	var Avalbytes []byte
	var dat map[string]interface{}
	layout := "2006-01-02 15:04:05"
	highestTime, err := time.Parse(layout, layout)

	for i := 0; i < len(rows); i++ {
		currentBid := rows[i].Columns[nCol].GetBytes()
		if err := json.Unmarshal(currentBid, &dat); err != nil {
			fmt.Println("GetHighestBid() Failed : Ummarshall error")
			return nil, fmt.Errorf("GetHighestBid(0 operation failed. %s", err)
		}
		bidTime, err := time.Parse(layout, dat["BidTime"].(string))
		if err != nil {
			fmt.Println("GetLastBid() Failed : time Conversion error on BidTime")
			return nil, fmt.Errorf("GetHighestBid() Int Conversion error on BidPrice! failed. %s", err)
		}

		if bidTime.Sub(highestTime) > 0 {
			highestTime = bidTime
			Avalbytes = currentBid
		}
	}

	return Avalbytes, nil

}

////////////////////////////////////////////////////////////////////////////
// Get The Highest Bid Received so far for an Auction
// in the block-chain
////////////////////////////////////////////////////////////////////////////
func GetNoOfBidsReceived(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	tn := "BidTable"
	rows, err := GetList(stub, tn, args)
	if err != nil {
		return nil, fmt.Errorf("GetLastBid operation failed. %s", err)
	}
	nBids := len(rows)
	return []byte(strconv.Itoa(nBids)), nil
}

////////////////////////////////////////////////////////////////////////////
// Get the Highest Bid in the List
//
////////////////////////////////////////////////////////////////////////////
func GetHighestBid(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {

	tn := "BidTable"
	rows, err := GetList(stub, tn, args)
	if err != nil {
		return nil, fmt.Errorf("GetLastBid operation failed. %s", err)
	}
	nCol := GetNumberOfKeys(tn)
	var Avalbytes []byte
	var dat map[string]interface{}
	var bidPrice, highestBid int
	highestBid = 0

	for i := 0; i < len(rows); i++ {
		currentBid := rows[i].Columns[nCol].GetBytes()
		if err := json.Unmarshal(currentBid, &dat); err != nil {
			fmt.Println("GetHighestBid() Failed : Ummarshall error")
			return nil, fmt.Errorf("GetHighestBid(0 operation failed. %s", err)
		}
		bidPrice, err = strconv.Atoi(dat["BidPrice"].(string))
		if err != nil {
			fmt.Println("GetHighestBid() Failed : Int Conversion error on BidPrice")
			return nil, fmt.Errorf("GetHighestBid() Int Conversion error on BidPrice! failed. %s", err)
		}

		if bidPrice >= highestBid {
			highestBid = bidPrice
			Avalbytes = currentBid
		}
	}

	return Avalbytes, nil
}

/////////////////////////////////////////////////////////////////
// This function checks the incoming args stuff for a valid record
// type entry as per the declared array recType[]
// The assumption is that rectType can be anywhere in the args or struct
// not necessarily in args[1] as per my old logic
// The Request type is used to process the record accordingly
/////////////////////////////////////////////////////////////////
func IdentifyReqType(args []string) string {
	for _, rt := range args {
		for _, val := range recType {
			if val == rt {
				return rt
			}
		}
	}
	return "DEFAULT"
}

/////////////////////////////////////////////////////////////////
// This function checks the incoming args stuff for a valid record
// type entry as per the declared array recType[]
// The assumption is that rectType can be anywhere in the args or struct
// not necessarily in args[1] as per my old logic
// The Request type is used to process the record accordingly
/////////////////////////////////////////////////////////////////
func ChkReqType(args []string) bool {
	for _, rt := range args {
		for _, val := range recType {
			if val == rt {
				return true
			}
		}
	}
	return false
}

/////////////////////////////////////////////////////////////////
// Checks if the incoming invoke has a valid requesType
// The Request type is used to process the record accordingly
// Old Logic (see new logic up)
/////////////////////////////////////////////////////////////////
func CheckRequestType(rt string) bool {
	for _, val := range recType {
		if val == rt {
			fmt.Println("CheckRequestType() : Valid Request Type , val : ", val, rt, "\n")
			return true
		}
	}
	fmt.Println("CheckRequestType() : Invalid Request Type , val : ", rt, "\n")
	return false
}

/////////////////////////////////////////////////////////////////////////////////////////////
// Return the right Object Buffer after validation to write to the ledger
// var recType = []string{"ARTINV", "USER", "BID", "AUCREQ", "POSTTRAN", "OPENAUC", "CLAUC"}
/////////////////////////////////////////////////////////////////////////////////////////////

func ProcessQueryResult(stub shim.ChaincodeStubInterface, Avalbytes []byte, args []string) error {

	// Identify Record Type by scanning the args for one of the recTypes
	// This is kind of a post-processor once the query fetches the results
	// RecType is the style of programming in the punch card days ..
	// ... well

	var dat map[string]interface{}

	if err := json.Unmarshal(Avalbytes, &dat); err != nil {
		panic(err)
	}

	var recType string
	recType = dat["RecType"].(string)
	switch recType {

	case "USER":
		ur, err := JSONtoUser(Avalbytes) //
		if err != nil {
			return err
		}
		fmt.Println("ProcessRequestType() : ", ur)
		return err

	case "CREATECONTR":
		ar, err := JSONtoAR(Avalbytes) //
		if err != nil {
			fmt.Println("ProcessRequestType(): Cannot create itemObject \n")
			return err
		}
		return err
		
	case "CLOSECONTRACT":
		ar, err := JSONtoAucReq(Avalbytes) //
		if err != nil {
			return err
		}
		fmt.Println("ProcessRequestType() : ", ar)
		return err
	case "CANCELCONTRACT":
	case "POSTTRAN":
		atr, err := JSONtoTran(Avalbytes) //
		if err != nil {
			return err
		}
		fmt.Println("ProcessRequestType() : ", atr)
		return err
	case "BID":
		bid, err := JSONtoBid(Avalbytes) //
		if err != nil {
			return err
		}
		fmt.Println("ProcessRequestType() : ", bid)
		return err
	case "DEFAULT":
		return nil
	case "XFER":
		return nil
	case "VERIFY":
		return nil
	default:

		return errors.New("Unknown")
	}
	return nil

}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Create a Command to execute Close Auction From the Command line
// cloaseauction.sh is created and then executed as seen below
// The file contains just one line
// /opt/gopath/src/github.com/hyperledger/fabric/peer chaincode invoke -l golang -n mycc -c '{"Function": "CloseAuction", "Args": ["1111","AUCREQ"]}'
// This approach has been used as opposed to exec.Command... because additional logic to gather environment variables etc. is required
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
func ShellCmdToCloseAuction(aucID string) error {
	gopath := os.Getenv("GOPATH")
	cdir := fmt.Sprintf("cd %s/src/github.com/hyperledger/fabric/", gopath)
	argStr := "'{\"Function\": \"CloseAuction\", \"Args\": [\"" + aucID + "\"," + "\"AUCREQ\"" + "]}'"
	argStr = fmt.Sprintf("%s/src/github.com/hyperledger/fabric/peer/peer chaincode invoke -l golang -n mycc -c %s", gopath, argStr)

	fileHandle, _ := os.Create(fmt.Sprintf("%s/src/github.com/hyperledger/fabric/peer/closeauction.sh", gopath))
	writer := bufio.NewWriter(fileHandle)
	defer fileHandle.Close()

	fmt.Fprintln(writer, cdir)
	fmt.Fprintln(writer, argStr)
	writer.Flush()

	x := "sh /opt/gopath/src/github.com/hyperledger/fabric/peer/closeauction.sh"
	err := exe_cmd(x)
	if err != nil {
		fmt.Println("%s", err)
	}

	err = exe_cmd("rm /opt/gopath/src/github.com/hyperledger/fabric/peer/closeauction.sh")
	if err != nil {
		fmt.Println("%s", err)
	}

	fmt.Println("Kicking off CloseAuction", argStr)
	return nil
}

func exe_cmd(cmd string) error {

	fmt.Println("command :  ", cmd)
	parts := strings.Fields(cmd)
	head := parts[0]
	parts = parts[1:len(parts)]

	_, err := exec.Command(head, parts...).CombinedOutput()
	if err != nil {
		fmt.Println("%s", err)
	}
	return err
}

//////////////////////////////////////////////////////////////////////////
// Update the Auction Object
// This function updates the status of the auction
// from INIT to OPEN to CLOSED
//////////////////////////////////////////////////////////////////////////

func UpdateContractStatus(stub shim.ChaincodeStubInterface, tableName string, ar AuctionRequest) ([]byte, error) {

	buff, err := AucReqtoJSON(ar)
	if err != nil {
		fmt.Println("UpdateContractStatus() : Failed Cannot create object buffer for write : ", ar.AuctionID)
		return nil, errors.New("UpdateContractStatus(): Failed Cannot create object buffer for write : " + ar.AuctionID)
	}

	// Update the ledger with the Buffer Data
	keys := []string{ar.AuctionID, ar.ItemID}
	err = ReplaceLedgerEntry(stub, "ContractTable", keys, buff)
	if err != nil {
		fmt.Println("UpdateContractStatus() : write error while inserting record\n")
		return buff, err
	}
	return buff, err
}

func main() {
	// maximize CPU usage for maximum performance
	//runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println("Starting Services: global job matching marketPlace ")

	gopath = os.Getenv("GOPATH")
	if len(os.Args) == 2 && strings.EqualFold(os.Args[1], "DEV") {
		fmt.Println("----------------- STARTED IN DEV MODE -------------------- ")
		//set chaincode path for DEV MODE
		ccPath = fmt.Sprintf("%s/src/github.com/hyperledger/fabric/service/global/", gopath)
	} else {
		fmt.Println("----------------- STARTED IN NET MODE -------------------- ")
		//set chaincode path for NET MODE
		ccPath = fmt.Sprintf("%s/src/github.com/Global-Blockchain/service/global/", gopath)
	}

	// Start the shim -- running the fabric
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}