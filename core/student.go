package core

import (
	"encoding/gob"
)

type StudentTxType byte

const (
	StudentTxTypeCreate StudentTxType = iota
	StudentTxTypeUpdate
	StudentTxTypeDelete
)

type Student struct {
	ID        string
	PaidDues  bool
	Semesters []Semester
	CGPA      float64
}

type Semester struct {
	SemesterNumber int
	SGPA           float64
}

type StudentTx struct {
	Type      StudentTxType
	StudentID string
	Student   *Student
	Fee       int64
}

func (s *Student) CalculateCGPA() {
	if len(s.Semesters) == 0 {
		s.CGPA = 0.0
		return
	}

	var totalSGPA float64
	for _, sem := range s.Semesters {
		totalSGPA += sem.SGPA
	}
	s.CGPA = totalSGPA / float64(len(s.Semesters))
}

func init() {
	gob.Register(StudentTx{})
	gob.Register(Student{})
	gob.Register(Semester{})
}
