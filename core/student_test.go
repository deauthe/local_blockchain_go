package core

import (
	"bytes"
	"encoding/gob"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deauthe/local_blockchain_go/crypto"
	"github.com/stretchr/testify/assert"
)

// Mock server to capture HTTP requests
func setupMockServer() (*httptest.Server, *[]*Transaction) {
	capturedTxs := &[]*Transaction{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/tx" {
			var tx Transaction
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
func testCreateStudent(privKey crypto.PrivateKey, studentID string, paidDues bool, semesters []Semester) error {
	student := &Student{
		ID:        studentID,
		PaidDues:  paidDues,
		Semesters: semesters,
	}
	student.CalculateCGPA()

	return testCreateStudentTx(privKey, studentID, student, StudentTxTypeCreate)
}

func testUpdateStudent(privKey crypto.PrivateKey, studentID string, paidDues bool, semesters []Semester) error {
	student := &Student{
		ID:        studentID,
		PaidDues:  paidDues,
		Semesters: semesters,
	}
	student.CalculateCGPA()

	return testCreateStudentTx(privKey, studentID, student, StudentTxTypeUpdate)
}

func testDeleteStudent(privKey crypto.PrivateKey, studentID string) error {
	return testCreateStudentTx(privKey, studentID, nil, StudentTxTypeDelete)
}

func testCreateStudentTx(privKey crypto.PrivateKey, studentID string, student *Student, txType StudentTxType) error {
	tx := NewTransaction(nil)
	tx.TxInner = StudentTx{
		Type:      txType,
		StudentID: studentID,
		Student:   student,
		Fee:       100, // Fixed fee for student transactions
	}

	if err := tx.Sign(privKey); err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	if err := tx.Encode(NewGobTxEncoder(buf)); err != nil {
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

func TestCreateStudent(t *testing.T) {
	// Setup mock server
	server, capturedTxs := setupMockServer()
	defer server.Close()

	// Create test data
	privKey := crypto.GeneratePrivateKey()
	studentID := "TEST001"
	paidDues := true
	semesters := []Semester{
		{SemesterNumber: 1, SGPA: 3.5},
		{SemesterNumber: 2, SGPA: 3.8},
	}

	// Call the function
	err := testCreateStudent(privKey, studentID, paidDues, semesters)

	// Verify results
	assert.NoError(t, err)
	assert.Len(t, *capturedTxs, 1)

	tx := (*capturedTxs)[0]
	studentTx, ok := tx.TxInner.(StudentTx)
	assert.True(t, ok)
	assert.Equal(t, StudentTxTypeCreate, studentTx.Type)
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
	semesters := []Semester{
		{SemesterNumber: 1, SGPA: 3.5},
		{SemesterNumber: 2, SGPA: 3.8},
		{SemesterNumber: 3, SGPA: 4.0},
	}

	// Call the function
	err := testUpdateStudent(privKey, studentID, paidDues, semesters)

	// Verify results
	assert.NoError(t, err)
	assert.Len(t, *capturedTxs, 1)

	tx := (*capturedTxs)[0]
	studentTx, ok := tx.TxInner.(StudentTx)
	assert.True(t, ok)
	assert.Equal(t, StudentTxTypeUpdate, studentTx.Type)
	assert.Equal(t, studentID, studentTx.StudentID)
	assert.Equal(t, int64(100), studentTx.Fee)

	// Verify student data
	assert.NotNil(t, studentTx.Student)
	assert.Equal(t, studentID, studentTx.Student.ID)
	assert.Equal(t, paidDues, studentTx.Student.PaidDues)
	assert.Len(t, studentTx.Student.Semesters, 3)
	assert.Equal(t, 3.5, studentTx.Student.Semesters[0].SGPA)
	assert.Equal(t, 3.8, studentTx.Student.Semesters[1].SGPA)
	assert.Equal(t, 4.0, studentTx.Student.Semesters[2].SGPA)
	assert.Equal(t, 3.77, studentTx.Student.CGPA) // (3.5 + 3.8 + 4.0) / 3
}

func TestDeleteStudent(t *testing.T) {
	// Setup mock server
	server, capturedTxs := setupMockServer()
	defer server.Close()

	// Create test data
	privKey := crypto.GeneratePrivateKey()
	studentID := "TEST003"

	// Call the function
	err := testDeleteStudent(privKey, studentID)

	// Verify results
	assert.NoError(t, err)
	assert.Len(t, *capturedTxs, 1)

	tx := (*capturedTxs)[0]
	studentTx, ok := tx.TxInner.(StudentTx)
	assert.True(t, ok)
	assert.Equal(t, StudentTxTypeDelete, studentTx.Type)
	assert.Equal(t, studentID, studentTx.StudentID)
	assert.Equal(t, int64(100), studentTx.Fee)
	assert.Nil(t, studentTx.Student) // Student should be nil for delete operations
}

func TestCreateStudentTx(t *testing.T) {
	// Setup mock server
	server, capturedTxs := setupMockServer()
	defer server.Close()

	// Create test data
	privKey := crypto.GeneratePrivateKey()
	studentID := "TEST004"
	student := &Student{
		ID:       studentID,
		PaidDues: true,
		Semesters: []Semester{
			{SemesterNumber: 1, SGPA: 3.5},
		},
	}
	student.CalculateCGPA()

	// Test create transaction
	err := testCreateStudentTx(privKey, studentID, student, StudentTxTypeCreate)
	assert.NoError(t, err)
	assert.Len(t, *capturedTxs, 1)

	tx := (*capturedTxs)[0]
	studentTx, ok := tx.TxInner.(StudentTx)
	assert.True(t, ok)
	assert.Equal(t, StudentTxTypeCreate, studentTx.Type)
	assert.Equal(t, studentID, studentTx.StudentID)
	assert.Equal(t, student, studentTx.Student)

	// Verify transaction was signed
	assert.NotNil(t, tx.Signature)
	assert.Equal(t, privKey.PublicKey(), tx.From)

	// Clear captured transactions
	*capturedTxs = []*Transaction{}

	// Test update transaction
	err = testCreateStudentTx(privKey, studentID, student, StudentTxTypeUpdate)
	assert.NoError(t, err)
	assert.Len(t, *capturedTxs, 1)

	tx = (*capturedTxs)[0]
	studentTx, ok = tx.TxInner.(StudentTx)
	assert.True(t, ok)
	assert.Equal(t, StudentTxTypeUpdate, studentTx.Type)

	// Clear captured transactions
	*capturedTxs = []*Transaction{}

	// Test delete transaction
	err = testCreateStudentTx(privKey, studentID, nil, StudentTxTypeDelete)
	assert.NoError(t, err)
	assert.Len(t, *capturedTxs, 1)

	tx = (*capturedTxs)[0]
	studentTx, ok = tx.TxInner.(StudentTx)
	assert.True(t, ok)
	assert.Equal(t, StudentTxTypeDelete, studentTx.Type)
	assert.Nil(t, studentTx.Student)
}
