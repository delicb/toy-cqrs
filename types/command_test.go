package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/delicb/toy-cqrs/types"
)

func TestCommandMarshalUnmarshal(t *testing.T) {
	originalPayload := map[string]interface{}{
		"foo": "bar",
		"baz": 42.0, // adding decimal point, since json is loosing int vs float information, so not to confuse tests
	}
	cmd, err := types.NewCommand("test", originalPayload)
	require.NoError(t, err)

	rawData, err := cmd.Marshal()
	require.NoError(t, err)

	newCmd := new(types.BaseCmd)
	require.NoError(t, newCmd.Unmarshal(rawData))
	var s map[string]interface{}
	require.NoError(t, newCmd.LoadPayload(&s))
	require.Equal(t, s, originalPayload)
	t.Log(s)
}
