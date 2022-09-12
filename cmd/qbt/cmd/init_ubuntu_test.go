package cmd

import (
	"github.com/stretchr/testify/assert"
	"os/exec"
	"strings"
	"testing"
)

func TestExecute(t *testing.T) {
	c := exec.Command("ls", "/")
	t.Log(c)
	out, err := c.Output()
	t.Log(err)
	assert.Nil(t, err)
	t.Log(string(out))
	assert.True(t, len(out) > 10)
	assert.True(t, strings.Contains(string(out), "tmp"))
}
