package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/deauthe/local_blockchain_go/core"
	"github.com/deauthe/local_blockchain_go/types"
)

func main() {
	// Define command line flags
	listHashes := flag.Bool("list", false, "List all block hashes")
	blockHash := flag.String("hash", "", "Hash of the block to inspect")
	txHash := flag.String("tx", "", "Hash of the transaction to inspect")
	pretty := flag.Bool("pretty", false, "Pretty print JSON output")
	flag.Parse()

	// Create persistent store
	store, err := core.NewPersistentStore()
	if err != nil {
		fmt.Printf("Error creating store: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// List all block hashes
	if *listHashes {
		hashes := store.GetAllBlockHashes()
		fmt.Println("Block hashes:")
		for hash, height := range hashes {
			fmt.Printf("Height: %d, Hash: %s\n", height, hash)
		}
		return
	}

	// Inspect specific transaction
	if *txHash != "" {
		hash := types.HashFromString(*txHash)
		// Search through all blocks to find the transaction
		blockHashes := store.GetAllBlockHashes()
		for blockHash := range blockHashes {
			block, err := store.GetBlockByHash(blockHash)
			if err != nil {
				continue
			}
			for _, tx := range block.Transactions {
				fmt.Print("considering tx:", tx.TxHash)
				if tx.TxHash == hash {
					// Convert transaction to JSON
					var jsonData []byte
					if *pretty {
						jsonData, err = json.MarshalIndent(tx, "", "  ")
					} else {
						jsonData, err = json.Marshal(tx)
					}
					if err != nil {
						fmt.Printf("Error marshaling transaction: %v\n", err)
						os.Exit(1)
					}

					fmt.Println("Transaction details:")
					fmt.Println(string(jsonData))

					// Print transaction inner data if it exists
					if tx.TxInner != nil {
						fmt.Println("\nInner Transaction Details:")
						switch inner := tx.TxInner.(type) {
						case core.StudentTx:
							fmt.Printf("  Student ID: %s\n", inner.StudentID)
							fmt.Printf("  Transaction Type: %s\n", inner.Type)
							if inner.Student != nil {
								fmt.Printf("  Student Details:\n")
								fmt.Printf("    ID: %s\n", inner.Student.ID)
								fmt.Printf("    Paid Dues: %v\n", inner.Student.PaidDues)
								fmt.Printf("    Semesters: %d\n", len(inner.Student.Semesters))
								for j, sem := range inner.Student.Semesters {
									fmt.Printf("      Semester %d: SGPA %.2f\n", j+1, sem.SGPA)
								}
							}
						case core.CollectionTx:
							fmt.Printf("  Fee: %d\n", inner.Fee)
							fmt.Printf("  MetaData: %s\n", string(inner.MetaData))
						case core.MintTx:
							fmt.Printf("  Fee: %d\n", inner.Fee)
							fmt.Printf("  NFT Hash: %s\n", inner.NFT)
							fmt.Printf("  Collection Hash: %s\n", inner.Collection)
							fmt.Printf("  MetaData: %s\n", string(inner.MetaData))
							fmt.Printf("  Collection Owner: %s\n", inner.CollectionOwner)
						}
					}
					return
				}
			}
		}
		fmt.Printf("Transaction with hash %s not found\n", hash)
		return
	}

	// Inspect specific block
	if *blockHash != "" {
		hash := types.HashFromString(*blockHash)
		block, err := store.GetBlockByHash(hash)
		if err != nil {
			fmt.Printf("Error getting block: %v\n", err)
			os.Exit(1)
		}

		// Convert block to JSON
		var jsonData []byte
		if *pretty {
			jsonData, err = json.MarshalIndent(block, "", "  ")
		} else {
			jsonData, err = json.Marshal(block)
		}
		if err != nil {
			fmt.Printf("Error marshaling block: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Block details:")
		fmt.Println(string(jsonData))

		// Print transaction details
		if len(block.Transactions) > 0 {
			fmt.Println("\nTransactions:")
			for i, tx := range block.Transactions {
				fmt.Printf("\nTransaction %d:\n", i+1)
				fmt.Printf("  From: %s\n", tx.From)
				fmt.Printf("  To: %s\n", tx.To)
				fmt.Printf("  Value: %d\n", tx.Value)
				fmt.Printf("  Hash: %s\n", tx.Hash(core.TxHasher{}))

				// Print transaction inner data if it exists
				if tx.TxInner != nil {
					fmt.Printf("  Inner Transaction Type: %T\n", tx.TxInner)
					// Handle specific transaction types
					switch inner := tx.TxInner.(type) {
					case core.StudentTx:
						fmt.Printf("  Student ID: %s\n", inner.StudentID)
						fmt.Printf("  Transaction Type: %s\n", inner.Type)
						if inner.Student != nil {
							fmt.Printf("  Student Details:\n")
							fmt.Printf("    ID: %s\n", inner.Student.ID)
							fmt.Printf("    Paid Dues: %v\n", inner.Student.PaidDues)
							fmt.Printf("    Semesters: %d\n", len(inner.Student.Semesters))
							for j, sem := range inner.Student.Semesters {
								fmt.Printf("      Semester %d: SGPA %.2f\n", j+1, sem.SGPA)
							}
						}
					case core.CollectionTx:
						fmt.Printf("  Fee: %d\n", inner.Fee)
						fmt.Printf("  MetaData: %s\n", string(inner.MetaData))
					case core.MintTx:
						fmt.Printf("  Fee: %d\n", inner.Fee)
						fmt.Printf("  NFT Hash: %s\n", inner.NFT)
						fmt.Printf("  Collection Hash: %s\n", inner.Collection)
						fmt.Printf("  MetaData: %s\n", string(inner.MetaData))
						fmt.Printf("  Collection Owner: %s\n", inner.CollectionOwner)
					}
				}
			}
		}
		return
	}

	// If no flags provided, show usage
	flag.Usage()
}
