package dotfile

import (
	"bytes"
	"errors"
)

const (
	testMessage      = "Test commit message"
	testHash         = "9abdbcf4ea4e2c1c077c21b8c2f2470ff36c31ce"
	testDirtyContent = "test content!\nanother line!\n"
)

type MockStorer struct {
	getTrackedErr       bool
	testAliasNotTracked bool
	dirtyContentErr     bool
	saveTrackedErr      bool
	revisionErr         bool
	uncompressErr       bool
	saveCommitErr       bool
	revertErr           bool
	hasCommit           bool
	hasCommitErr        bool
}

func (ms *MockStorer) HasCommit(string) (bool, error) {
	if ms.hasCommitErr {
		return false, errors.New("has commit error")
	}
	return ms.hasCommit, nil
}

func (ms *MockStorer) DirtyContent() ([]byte, error) {
	if ms.dirtyContentErr {
		return nil, errors.New("get contents error")
	}
	return []byte(testDirtyContent), nil
}

func (ms *MockStorer) Revision(string) ([]byte, error) {
	if ms.revisionErr {
		return nil, errors.New("revision error")
	}
	if ms.uncompressErr {
		return nil, nil
	}

	compressed, _, err := hashAndCompress([]byte(testDirtyContent))
	if err != nil {
		return nil, err
	}

	return compressed.Bytes(), nil
}

func (ms *MockStorer) SaveCommit(*bytes.Buffer, *Commit) error {
	if ms.saveCommitErr {
		return errors.New("save revision error")
	}
	return nil
}

func (ms *MockStorer) Revert(*bytes.Buffer, string) error {
	if ms.revertErr {
		return errors.New("revert error")
	}

	return nil
}
