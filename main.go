package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/deauthe/local_blockchain_go/core"
	"github.com/deauthe/local_blockchain_go/crypto"
	"github.com/deauthe/local_blockchain_go/mock"
	"github.com/deauthe/local_blockchain_go/network"
	"github.com/deauthe/local_blockchain_go/util"
	"github.com/go-kit/log"
)

func main() {
	// Initialize logger
	logger, err := util.NewLogger("blockchain")
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// Create validator node with private key
	validatorPrivKey := crypto.GeneratePrivateKey()
	localNode := makeServer("LOCAL_NODE", &validatorPrivKey, ":3000", []string{":4000"}, ":9000", logger)
	go localNode.Start()

	// Create remote nodes for network
	remoteNode := makeServer("REMOTE_NODE_B", nil, ":4000", []string{":5000"}, "", logger)
	go remoteNode.Start()

	remoteNodeB := makeServer("REMOTE_NODE", nil, ":5000", nil, "", logger)
	go remoteNodeB.Start()

	// Wait for network to stabilize
	time.Sleep(2 * time.Second)

	logger.Log("msg", "Starting student operations demonstration...")

	// Create multiple students with different semesters

	// Create students
	logger.Log("msg", "Creating students...")
	for _, student := range mock.MockStudents {
		startTime := time.Now()
		logger.Log("msg", fmt.Sprintf("Creating student %s...", student.Id))
		if err := createStudent(validatorPrivKey, student.Id, student.PaidDues, student.Semesters); err != nil {
			logger.Log("msg", fmt.Sprintf("Error creating student %s: %v (took %v)", student.Id, err, time.Since(startTime)))
			os.Exit(1)
		}
		util.LogWithTiming(logger, startTime, "Created student %s", student.Id)
		time.Sleep(1 * time.Second) // Wait for transaction to be processed
	}

	// Update students
	logger.Log("msg", "Updating students...")
	for _, student := range mock.MockStudents {
		startTime := time.Now()
		// Add a new semester to each student
		updatedSemesters := append(student.Semesters, core.Semester{
			SemesterNumber: len(student.Semesters) + 1,
			SGPA:           4.0,
		})

		logger.Log("msg", fmt.Sprintf("Updating student %s with new semester...", student.Id))
		if err := updateStudent(validatorPrivKey, student.Id, student.PaidDues, updatedSemesters); err != nil {
			logger.Log("msg", fmt.Sprintf("Error updating student %s: %v (took %v)", student.Id, err, time.Since(startTime)))
			os.Exit(1)
		}
		util.LogWithTiming(logger, startTime, "Updated student %s", student.Id)
		time.Sleep(1 * time.Second) // Wait for transaction to be processed
	}

	// Delete one student
	logger.Log("msg", "Deleting a student...")
	startTime := time.Now()
	if err := deleteStudent(validatorPrivKey, mock.MockStudents[0].Id); err != nil {
		logger.Log("msg", fmt.Sprintf("Error deleting student %s: %v (took %v)", mock.MockStudents[0].Id, err, time.Since(startTime)))
		os.Exit(1)
	}
	util.LogWithTiming(logger, startTime, "Deleted student %s", mock.MockStudents[0].Id)

	// Keep the program running to see the blockchain state
	logger.Log("msg", "Student operations completed. Blockchain is running...")
	logger.Log("msg", "Check the blockchain state through the API endpoints.")
	select {}
}

func makeServer(id string, pk *crypto.PrivateKey, addr string, seedNodes []string, apiListenAddr string, logger log.Logger) *network.Server {
	opts := network.ServerOpts{
		APIListenAddr: apiListenAddr,
		SeedNodes:     seedNodes,
		ListenAddr:    addr,
		PrivateKey:    pk,
		ID:            id,
		Logger:        logger,
	}

	s, err := network.NewServer(opts)
	if err != nil {
		logger.Log("msg", fmt.Sprintf("Error creating server %s: %v", id, err))
		os.Exit(1)
	}

	return s
}

func createStudentTx(privKey crypto.PrivateKey, studentID string, student *core.Student, txType core.StudentTxType) error {
	tx := core.NewTransaction(nil)
	tx.TxInner = core.StudentTx{
		Type:      txType,
		StudentID: studentID,
		Student:   student,
		Fee:       100, // Fixed fee for student transactions
	}

	// Set the From field before signing
	tx.From = privKey.PublicKey()

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

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody := new(bytes.Buffer)
	respBody.ReadFrom(resp.Body)

	// Log status and body
	fmt.Printf("[HTTP %s] %s\n", resp.Status, respBody.String())

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response: %s", resp.Status)
	}

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
