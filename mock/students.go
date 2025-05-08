package mock

import "github.com/deauthe/local_blockchain_go/core"

var MockStudents = []struct {
	Id        string
	PaidDues  bool
	Semesters []core.Semester
}{
	{
		Id:       "STU001",
		PaidDues: true,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.5},
			{SemesterNumber: 2, SGPA: 3.8},
		},
	},
	{
		Id:       "STU002",
		PaidDues: true,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.2},
			{SemesterNumber: 2, SGPA: 3.6},
			{SemesterNumber: 3, SGPA: 3.9},
		},
	},
	{
		Id:       "STU003",
		PaidDues: true,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.2},
			{SemesterNumber: 2, SGPA: 3.6},
			{SemesterNumber: 3, SGPA: 3.9},
		},
	},
	{
		Id:       "STU004",
		PaidDues: false,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.2},
			{SemesterNumber: 2, SGPA: 3.6},
			{SemesterNumber: 3, SGPA: 3.9},
		},
	},
	{
		Id:       "STU005",
		PaidDues: true,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.2},
			{SemesterNumber: 2, SGPA: 3.6},
			{SemesterNumber: 3, SGPA: 3.9},
		},
	},
	{
		Id:       "STU006",
		PaidDues: true,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.2},
			{SemesterNumber: 2, SGPA: 3.6},
			{SemesterNumber: 3, SGPA: 3.9},
		},
	},
	{
		Id:       "STU007",
		PaidDues: true,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.2},
			{SemesterNumber: 2, SGPA: 3.6},
			{SemesterNumber: 3, SGPA: 3.9},
		},
	},
	{
		Id:       "STU008",
		PaidDues: true,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.2},
			{SemesterNumber: 2, SGPA: 3.6},
			{SemesterNumber: 3, SGPA: 3.9},
		},
	},
	{
		Id:       "STU009",
		PaidDues: true,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.2},
			{SemesterNumber: 2, SGPA: 3.6},
			{SemesterNumber: 3, SGPA: 3.9},
		},
	},
	{
		Id:       "STU0010",
		PaidDues: true,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.2},
			{SemesterNumber: 2, SGPA: 3.6},
			{SemesterNumber: 3, SGPA: 3.9},
		},
	},
	{
		Id:       "STU0011",
		PaidDues: true,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.2},
			{SemesterNumber: 2, SGPA: 3.6},
			{SemesterNumber: 3, SGPA: 3.9},
		},
	},
	{
		Id:       "STU0012",
		PaidDues: true,
		Semesters: []core.Semester{
			{SemesterNumber: 1, SGPA: 3.2},
			{SemesterNumber: 2, SGPA: 3.6},
			{SemesterNumber: 3, SGPA: 3.9},
		},
	},
}
