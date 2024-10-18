package lib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ValidateLogsGlobalServices(t *testing.T, logsList []string, contexts []string) map[string]int {
	logsMap := Uniq(logsList)
	t.Log("Value of logs is:", MapToString(logsMap))
	require.Equal(t, len(logsMap), len(contexts))
	for _, c := range contexts {
		require.Contains(t, logsList, c)
	}
	return logsMap
}

func ValidateLogsDB(t *testing.T, logsList []string, expectedContext string) map[string]int {
	logsMap := Uniq(logsList)
	t.Log("Value of logs is:", MapToString(logsMap))
	require.Contains(t, logsList, expectedContext)
	require.Equal(t, len(logsMap), 1)
	return logsMap
}

func ValidateLogsSharedStep1(t *testing.T, logsList []string, context string, expectedContexts []string) map[string]int {
	logsMap := Uniq(logsList)
	t.Log("Value of logs is:", MapToString(logsMap))
	expectedCount := 2
	if context != expectedContexts[1] {
		expectedContexts = []string{expectedContexts[0]}
		expectedCount = 1
	}
	t.Logf("Context: %s, Expected Contexts: %v, Expected Count: %d", context, expectedContexts, expectedCount)
	require.Equal(t, len(logsMap), expectedCount)
	for _, c := range expectedContexts {
		require.Contains(t, logsList, c)
	}
	return logsMap
}

func ValidateLogsSharedStep2(t *testing.T, logsList []string, context string, expectedContexts []string) map[string]int {
	logsMap := Uniq(logsList)
	t.Log("Value of logs is:", MapToString(logsMap))
	expectedCount := 2
	if context != expectedContexts[0] {
		expectedContexts = []string{expectedContexts[1]}
		expectedCount = 1
	}
	t.Logf("Context: %s, Expected Contexts: %v, Expected Count: %d", context, expectedContexts, expectedCount)
	require.Equal(t, len(logsMap), expectedCount)
	for _, c := range expectedContexts {
		require.Contains(t, logsList, c)
	}
	return logsMap
}
