package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPool_Close_NilLoggerPool(t *testing.T) {
	p := &Pool{Pool: nil, logger: nil}
	assert.NotPanics(t, func() {
		p.Close()
	})
}

func TestPool_Close_RealPool_NilLogger(t *testing.T) {
	p := &Pool{Pool: nil, logger: nil}
	assert.NotPanics(t, func() {
		p.Close()
	})
}
