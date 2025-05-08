package core_test

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deauthe/local_blockchain_go/core"
	"github.com/deauthe/local_blockchain_go/crypto"
	"github.com/stretchr/testify/assert"
)

// Mock server to capture HTTP requests
func setupMockServer() (*httptest.Server, *[]*core.Transaction) {
	capturedTxs := &[]*core.Transaction{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/tx" {
			var tx core.Transaction
			decoder := gob.NewDecoder(r.Body)
			err := decoder.Decode(&tx)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			*capturedTxs = append(*capturedTxs, &tx)
			w.WriteHeader(http.StatusOK)
		} else {
			http.NotFound(w, r)
		}
	}))

	return server, capturedTxs
}

// Test helper functions that mirror the actual functions
func testCreateStudent(server *httptest.Server, privKey crypto.PrivateKey, studentID string, paidDues bool, semesters []core.Semester) error {
	student := &core.Student{
		ID:        studentID,
		PaidDues:  paidDues,
		Semesters: semesters,
	}
	student.CalculateCGPA()

	return testCreateStudentTx(server, privKey, studentID, student, core.StudentTxTypeCreate)
}

func testUpdateStudent(server *httptest.Server, privKey crypto.PrivateKey, studentID string, paidDues bool, semesters []core.Semester) error {
	student := &core.Student{
		ID:        studentID,
		PaidDues:  paidDues,
		Semesters: semesters,
	}
	student.CalculateCGPA()

	return testCreateStudentTx(server, privKey, studentID, student, core.StudentTxTypeUpdate)
}

func testDeleteStudent(server *httptest.Server, privKey crypto.PrivateKey, studentID string) error {
	return testCreateStudentTx(server, privKey, studentID, nil, core.StudentTxTypeDelete)
}

func testCreateStudentTx(server *httptest.Server, privKey crypto.PrivateKey, studentID string, student *core.Student, txType core.StudentTxType) error {
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

	req, err := http.NewRequest("POST", server.URL+"/tx", buf)
	if err != nil {
		return err
	}

	client := http.Client{}
	_, err = client.Do(req)

	return err
}

func TestCreateStudent(t *testing.T) {
	// Setup mock server
	server, capturedTxs := setupMockServer()
	defer server.Close()

	// Create test data
	privKey := crypto.GeneratePrivateKey()
	studentID := "TEST001"
	paidDues := true
	semesters := []core.Semester{
		{SemesterNumber: 1, SGPA: 3.5},
		{SemesterNumber: 2, SGPA: 3.8},
	}

	// Call the function
	err := testCreateStudent(server, privKey, studentID, paidDues, semesters)

	// Verify results
	assert.NoError(t, err)
	assert.Len(t, *capturedTxs, 1)

	tx := (*capturedTxs)[0]
	studentTx, ok := tx.TxInner.(core.StudentTx)
	assert.True(t, ok)
	assert.Equal(t, core.StudentTxTypeCreate, studentTx.Type)
	assert.Equal(t, studentID, studentTx.StudentID)
	assert.Equal(t, int64(100), studentTx.Fee)

	// Verify student data
	assert.NotNil(t, studentTx.Student)
	assert.Equal(t, studentID, studentTx.Student.ID)
	assert.Equal(t, paidDues, studentTx.Student.PaidDues)
	assert.Len(t, studentTx.Student.Semesters, 2)
	assert.Equal(t, 3.5, studentTx.Student.Semesters[0].SGPA)
	assert.Equal(t, 3.8, studentTx.Student.Semesters[1].SGPA)
	assert.Equal(t, 3.65, studentTx.Student.CGPA) // (3.5 + 3.8) / 2
}

func TestUpdateStudent(t *testing.T) {
	// Setup mock server
	server, capturedTxs := setupMockServer()
	defer server.Close()

	// Create test data
	privKey := crypto.GeneratePrivateKey()
	studentID := "TEST002"
	paidDues := true
	semesters := []core.Semester{
		{SemesterNumber: 1, SGPA: 3.0},
		{SemesterNumber: 2, SGPA: 3.8},
	}

	// Call the function
	err := testUpdateStudent(server, privKey, studentID, paidDues, semesters)

	// Verify results
	assert.NoError(t, err)
	assert.Len(t, *capturedTxs, 1)

	tx := (*capturedTxs)[0]
	studentTx, ok := tx.TxInner.(core.StudentTx)
	assert.True(t, ok)
	assert.Equal(t, core.StudentTxTypeUpdate, studentTx.Type)
	assert.Equal(t, studentID, studentTx.StudentID)
	assert.Equal(t, int64(100), studentTx.Fee)

	// Verify student data
	assert.NotNil(t, studentTx.Student)
	assert.Equal(t, studentID, studentTx.Student.ID)
	assert.Equal(t, paidDues, studentTx.Student.PaidDues)
	assert.Len(t, studentTx.Student.Semesters, 2)
	assert.Equal(t, 3.0, studentTx.Student.Semesters[0].SGPA)
	assert.Equal(t, 3.8, studentTx.Student.Semesters[1].SGPA)
	assert.Equal(t, 3.4, studentTx.Student.CGPA) // (3.5 + 3.8 + 4.0) / 3

	fmt.Print("✅ updating student passed\n with data : ", studentTx.Student, "\n", "fee : ", studentTx.Fee, "\n", "tx : ", tx.Hash(core.TxHasher{}).String(), "\n")
}

func TestDeleteStudent(t *testing.T) {
	// Setup mock server
	server, capturedTxs := setupMockServer()
	defer server.Close()

	// Create test data
	privKey := crypto.GeneratePrivateKey()
	studentID := "TEST003"

	// Call the function
	err := testDeleteStudent(server, privKey, studentID)

	// Verify results
	assert.NoError(t, err)
	assert.Len(t, *capturedTxs, 1)

	tx := (*capturedTxs)[0]
	studentTx, ok := tx.TxInner.(core.StudentTx)
	assert.True(t, ok)
	assert.Equal(t, core.StudentTxTypeDelete, studentTx.Type)
	assert.Equal(t, studentID, studentTx.StudentID)
	assert.Equal(t, int64(100), studentTx.Fee)
	assert.Nil(t, studentTx.Student) // Student should be nil for delete operations

	fmt.Print("✅ deleting student passed\n with data : ", studentTx.Student, "\n", "fee : ", studentTx.Fee, "\n", "tx : ", tx, "\n")
}

func TestCreateStudentTx(t *testing.T) {
	// Setup mock server
	server, capturedTxs := setupMockServer()
	defer server.Close()

	// Create test data
	privKey := crypto.GeneratePrivateKey()
	studentID := "TEST004"
	student := &core.Student{
		ID:       studentID,
		PaidDues: true,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.5},
		},
	}
	student.CalculateCGPA()

	// Test create transaction
	err := testCreateStudentTx(server, privKey, studentID, student, core.StudentTxTypeCreate)
	assert.NoError(t, err)
	assert.Len(t, *capturedTxs, 1)

	tx := (*capturedTxs)[0]
	studentTx, ok := tx.TxInner.(core.StudentTx)
	assert.True(t, ok)
	assert.Equal(t, core.StudentTxTypeCreate, studentTx.Type)
	assert.Equal(t, studentID, studentTx.StudentID)
	assert.Equal(t, student, studentTx.Student)

	// Verify transaction was signed
	assert.NotNil(t, tx.Signature)
	assert.Equal(t, privKey.PublicKey(), tx.From)

	// Clear captured transactions
	*capturedTxs = []*core.Transaction{}

	// Test update transaction
	err = testCreateStudentTx(server, privKey, studentID, student, core.StudentTxTypeUpdate)
	assert.NoError(t, err)
	assert.Len(t, *capturedTxs, 1)

	tx = (*capturedTxs)[0]
	studentTx, ok = tx.TxInner.(core.StudentTx)
	assert.True(t, ok)
	assert.Equal(t, core.StudentTxTypeUpdate, studentTx.Type)

	// Clear captured transactions
	*capturedTxs = []*core.Transaction{}

	// Test delete transaction
	err = testCreateStudentTx(server, privKey, studentID, nil, core.StudentTxTypeDelete)
	assert.NoError(t, err)
	assert.Len(t, *capturedTxs, 1)

	tx = (*capturedTxs)[0]
	studentTx, ok = tx.TxInner.(core.StudentTx)
	assert.True(t, ok)
	assert.Equal(t, core.StudentTxTypeDelete, studentTx.Type)
	assert.Nil(t, studentTx.Student)
}
