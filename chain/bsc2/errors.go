package bsc2

import "errors"

var (
	ErrInconsistentBlock = errors.New("Inconsistent block")
	ErrRecoverable       = errors.New("RecoverableErr")
)

func recoverable(err error) error {
	if err == ErrInconsistentBlock {
		return ErrRecoverable
	}
	return err
}
