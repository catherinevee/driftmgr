package remediation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeletionEngine_NewEngine(t *testing.T) {
	engine := NewDeletionEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.providers)
}

