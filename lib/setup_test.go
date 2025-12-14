package lib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNodeServer(t *testing.T) {

	t.Run("1", func(t *testing.T) {
		got, err := GetNodeServer("../cmd/node01")
		fmt.Println(got, err)
		assert.NoError(t, err)
	})

}
