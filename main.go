package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/deauthe/local_blockchain_go/core"
	"github.com/deauthe/local_blockchain_go/crypto"
	"github.com/deauthe/local_blockchain_go/network"
	"github.com/deauthe/local_blockchain_go/types"
	"github.com/deauthe/local_blockchain_go/util"
)

func main() {
	validatorPrivKey := crypto.GeneratePrivateKey()
	localNode := makeServer("LOCAL_NODE", &validatorPrivKey, ":3000", []string{":4000"}, ":9000")
	go localNode.Start()

	remoteNode := makeServer("REMOTE_NODE", nil, ":4000", []string{":5000"}, "")
	go remoteNode.Start()

	remoteNodeB := makeServer("REMOTE_NODE_B", nil, ":5000", nil, "")
	go remoteNodeB.Start()

	go func() {
		time.Sleep(11 * time.Second)

		lateNode := makeServer("LATE_NODE", nil, ":6000", []string{":4000"}, "")
		go lateNode.Start()
	}()

	time.Sleep(1 * time.Second)

	// if err := sendTransaction(validatorPrivKey); err != nil {
	// 	panic(err)
	// }

	// collectionOwnerPrivKey := crypto.GeneratePrivateKey()
	// collectionHash := createCollectionTx(collectionOwnerPrivKey)

	// txSendTicker := time.NewTicker(1 * time.Second)
	// go func() {
	// 	for i := 0; i < 20; i++ {
	// 		nftMinter(collectionOwnerPrivKey, collectionHash)

	// 		<-txSendTicker.C
	// 	}
	// }()

	// Example: Create a student
	time.Sleep(2 * time.Second)

	// Create a student with 2 semesters
	semesters := []core.Semester{
		{SemesterNumber: 1, SGPA: 3.5},
		{SemesterNumber: 2, SGPA: 3.8},
	}

	if err := createStudent(validatorPrivKey, "STU001", true, semesters); err != nil {
		log.Fatal(err)
	}

	// Update student's semester data
	time.Sleep(2 * time.Second)
	semesters = append(semesters, core.Semester{SemesterNumber: 3, SGPA: 4.0})
	if err := updateStudent(validatorPrivKey, "STU001", true, semesters); err != nil {
		log.Fatal(err)
	}

	// Delete student
	time.Sleep(2 * time.Second)
	if err := deleteStudent(validatorPrivKey, "STU001"); err != nil {
		log.Fatal(err)
	}

	select {}
}

func sendTransaction(privKey crypto.PrivateKey) error {
	toPrivKey := crypto.GeneratePrivateKey()

	tx := core.NewTransaction(nil)
	tx.To = toPrivKey.PublicKey()
	tx.Value = 666

	if err := tx.Sign(privKey); err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	if err := tx.Encode(core.NewGobTxEncoder(buf)); err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", "http://localhost:9000/tx", buf)
	if err != nil {
		panic(err)
	}

	client := http.Client{}
	_, err = client.Do(req)

	return err
}

func makeServer(id string, pk *crypto.PrivateKey, addr string, seedNodes []string, apiListenAddr string) *network.Server {
	opts := network.ServerOpts{
		APIListenAddr: apiListenAddr,
		SeedNodes:     seedNodes,
		ListenAddr:    addr,
		PrivateKey:    pk,
		ID:            id,
	}

	s, err := network.NewServer(opts)
	if err != nil {
		log.Fatal(err)
	}

	return s
}

func createCollectionTx(privKey crypto.PrivateKey) types.Hash {
	tx := core.NewTransaction(nil)
	tx.TxInner = core.CollectionTx{
		Fee:      200,
		MetaData: []byte("chicken and egg collection!"),
	}
	tx.Sign(privKey)

	buf := &bytes.Buffer{}
	if err := tx.Encode(core.NewGobTxEncoder(buf)); err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", "http://localhost:9000/tx", buf)
	if err != nil {
		panic(err)
	}

	client := http.Client{}
	_, err = client.Do(req)
	if err != nil {
		panic(err)
	}

	return tx.Hash(core.TxHasher{})
}

func nftMinter(privKey crypto.PrivateKey, collection types.Hash) {
	metaData := map[string]any{
		"power":  8,
		"health": 100,
		"color":  "green",
		"rare":   "yes",
	}

	metaBuf := new(bytes.Buffer)
	if err := json.NewEncoder(metaBuf).Encode(metaData); err != nil {
		panic(err)
	}

	tx := core.NewTransaction(nil)
	tx.TxInner = core.MintTx{
		Fee:             200,
		NFT:             util.RandomHash(),
		MetaData:        metaBuf.Bytes(),
		Collection:      collection,
		CollectionOwner: privKey.PublicKey(),
	}
	tx.Sign(privKey)

	buf := &bytes.Buffer{}
	if err := tx.Encode(core.NewGobTxEncoder(buf)); err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", "http://localhost:9000/tx", buf)
	if err != nil {
		panic(err)
	}

	client := http.Client{}
	_, err = client.Do(req)
	if err != nil {
		panic(err)
	}
}

func createStudentTx(privKey crypto.PrivateKey, studentID string, student *core.Student, txType core.StudentTxType) error {
	tx := core.NewTransaction(nil)
	tx.TxInner = core.StudentTx{
		Type:      txType,
		StudentID: studentID,
		Student:   student,
		Fee:       100, // Fixed fee for student transactions
	}

	if err := tx.Sign(privKey); err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	if err := tx.Encode(core.NewGobTxEncoder(buf)); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "http://localhost:9000/tx", buf)
	if err != nil {
		return err
	}

	client := http.Client{}
	_, err = client.Do(req)

	return err
}

func createStudent(privKey crypto.PrivateKey, studentID string, paidDues bool, semesters []core.Semester) error {
	student := &core.Student{
		ID:        studentID,
		PaidDues:  paidDues,
		Semesters: semesters,
	}
	student.CalculateCGPA()

	return createStudentTx(privKey, studentID, student, core.StudentTxTypeCreate)
}

func updateStudent(privKey crypto.PrivateKey, studentID string, paidDues bool, semesters []core.Semester) error {
	student := &core.Student{
		ID:        studentID,
		PaidDues:  paidDues,
		Semesters: semesters,
	}
	student.CalculateCGPA()

	return createStudentTx(privKey, studentID, student, core.StudentTxTypeUpdate)
}

func deleteStudent(privKey crypto.PrivateKey, studentID string) error {
	return createStudentTx(privKey, studentID, nil, core.StudentTxTypeDelete)
}
