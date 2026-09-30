package sqlcleanup

import (
	"database/sql"
	"errors"
	"testing"
)

const testWrap = "roll back"

var (
	errTestRollback = errors.New("connection reset during rollback")
	errTestCause    = errors.New("insert failed")
	errTestClose    = errors.New("close failed")
)

type fakeTx struct{ err error }

func (f fakeTx) Rollback() error { return f.err }

type fakeRows struct{ err error }

func (f fakeRows) Close() error { return f.err }

func TestRollbackUncommittedSurfacesFailure(t *testing.T) {
	var err error
	RollbackUncommitted(fakeTx{err: errTestRollback}, testWrap, &err)
	if !errors.Is(err, errTestRollback) {
		t.Fatalf("rollback failure not surfaced: got %v", err)
	}
}

func TestRollbackUncommittedKeepsCause(t *testing.T) {
	err := errTestCause
	RollbackUncommitted(fakeTx{err: errTestRollback}, testWrap, &err)
	if !errors.Is(err, errTestCause) || !errors.Is(err, errTestRollback) {
		t.Fatalf("want cause and rollback failure joined, got %v", err)
	}
}

func TestRollbackUncommittedIgnoresCommittedTx(t *testing.T) {
	var err error
	RollbackUncommitted(fakeTx{err: sql.ErrTxDone}, testWrap, &err)
	if err != nil {
		t.Fatalf("sql.ErrTxDone after commit must be ignored, got %v", err)
	}
	err = errTestCause
	RollbackUncommitted(fakeTx{}, testWrap, &err)
	if err != errTestCause {
		t.Fatalf("successful rollback must keep the cause unchanged, got %v", err)
	}
}

func TestCloseRowsReportsFirstFailureOnly(t *testing.T) {
	var err error
	CloseRows(fakeRows{err: errTestClose}, testWrap, &err)
	if !errors.Is(err, errTestClose) {
		t.Fatalf("close failure not surfaced: got %v", err)
	}
	err = errTestCause
	CloseRows(fakeRows{err: errTestClose}, testWrap, &err)
	if err != errTestCause {
		t.Fatalf("an existing error must win over a close failure, got %v", err)
	}
}
