package ethereum

/*
frontend

// Using ethers.js to connect to MetaMask:
const provider = new ethers.providers.Web3Provider(window.ethereum);
const signer = provider.getSigner();
const contract = new ethers.Contract(contractAddress, contractABI, signer);

// When user updates site data:
const tx = await contract.setSite("mywebsite", "QmYourIPFSCID");
await tx.wait();
console.log("Transaction confirmed!", tx.hash);

*/

/*
 client (e.g., a web dApp using MetaMask) signs the transaction and sends the signed transaction as a hexadecimal string in the POST body. The Go backend reads this signed transaction, decodes it, unmarshals it into a go-ethereum Transaction object, and then sends
 it using the Avalanche RPC API.

 package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const rpcURL = "https://api.avax-test.network/ext/bc/C/rpc"

// sendTxHandler receives a signed transaction in hex format and broadcasts it.
func sendTxHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the signed transaction hex string from the request body.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	txHex := string(body)

	// Decode the hex string into bytes.
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		http.Error(w, "Invalid hex string", http.StatusBadRequest)
		return
	}

	// Unmarshal the bytes into a Transaction object.
	tx := new(types.Transaction)
	if err := tx.UnmarshalBinary(txBytes); err != nil {
		http.Error(w, "Failed to unmarshal transaction", http.StatusBadRequest)
		return
	}

	// Connect to the Avalanche RPC endpoint.
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		http.Error(w, "Failed to connect to Avalanche", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	// Broadcast the signed transaction.
	if err := client.SendTransaction(context.Background(), tx); err != nil {
		http.Error(w, fmt.Sprintf("Failed to send transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Respond with the transaction hash.
	fmt.Fprintf(w, "Transaction sent: %s", tx.Hash().Hex())
}

func main() {
	http.HandleFunc("/sendTx", sendTxHandler)
	log.Println("Backend listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
*/

/*
Solidity contract with small processing fee and address where fee should be sent

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract SiteRegistry {
    struct Site {
        string ipfsCID;
        address owner;
    }

    mapping(string => Site) public sites;

    // The platform's address that receives fees.
    address payable public platformAddress;

    // Fee to create or update a site (e.g., 0.001 AVAX)
    uint256 public constant CREATION_FEE = 1000000000000000;

    event SiteUpdated(string indexed name, string ipfsCID, address indexed owner);

    // Set the platform address in the constructor
    constructor(address payable _platformAddress) {
        platformAddress = _platformAddress;
    }

    // Payable function to create/update a site mapping with a fee.
    function setSite(string memory name, string memory ipfsCID) public payable {
        require(msg.value >= CREATION_FEE, "Insufficient fee");
        require(bytes(name).length > 0, "Site name cannot be empty");
        require(bytes(ipfsCID).length > 0, "IPFS CID cannot be empty");

        // If the site doesn't exist, register it; otherwise, only allow the owner to update.
        if (sites[name].owner == address(0)) {
            sites[name] = Site(ipfsCID, msg.sender);
        } else {
            require(msg.sender == sites[name].owner, "Only the owner can update this site");
            sites[name].ipfsCID = ipfsCID;
        }

        // Optionally, forward the fee to the platform's address:
        platformAddress.transfer(msg.value);

        emit SiteUpdated(name, ipfsCID, msg.sender);
    }

    function getSite(string memory name) public view returns (string memory) {
        return sites[name].ipfsCID;
    }
}



*/
