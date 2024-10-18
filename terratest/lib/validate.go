package lib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ValidateLogsAllValues(t *testing.T, logsList []string, contexts []string) map[string]int {
	logsMap := Uniq(logsList)
	t.Log("Value of logs is:", MapToString(logsMap))
	require.Equal(t, len(logsMap), len(contexts))
	for _, c := range contexts {
		require.Contains(t, logsList, c)
	}
	return logsMap
}

func ValidateLogsOnlyOneValue(t *testing.T, logsList []string, expectedContext string) map[string]int {
	logsMap := Uniq(logsList)
	t.Log("Value of logs is:", MapToString(logsMap))
	require.Contains(t, logsList, expectedContext)
	require.Equal(t, len(logsMap), 1)
	return logsMap
}
