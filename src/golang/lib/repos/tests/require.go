package tests

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aqueducthq/aqueduct/lib/models"
	"github.com/aqueducthq/aqueduct/lib/models/views"
	"github.com/stretchr/testify/require"
)

func requireDeepEqual(t *testing.T, expected, actual interface{}) {
	require.True(
		t,
		reflect.DeepEqual(
			expected,
			actual,
		),
		fmt.Sprintf("Expected: %v\n Actual: %v", expected, actual),
	)
}

// requireDeepEqualWorkflows asserts that the expected and actual lists of Workflows
// contain the same elements.
func requireDeepEqualWorkflows(t *testing.T, expected, actual []models.Workflow) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedWorkflow := range expected {
		found := false
		var foundWorkflow models.Workflow

		for _, actualWorkflow := range actual {
			if expectedWorkflow.ID == actualWorkflow.ID {
				found = true
				foundWorkflow = actualWorkflow
				break
			}
		}

		require.True(t, found, "Unable to find workflow: %v", expectedWorkflow)
		requireDeepEqual(t, expectedWorkflow, foundWorkflow)
	}
}

// requireDeepEqualIntegration asserts that the expected and actual lists of Integrations
// contain the same elements.
func requireDeepEqualIntegrations(t *testing.T, expected, actual []models.Integration) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedIntegration := range expected {
		found := false
		var foundIntegration models.Integration

		for _, actualIntegration := range actual {
			if expectedIntegration.ID == actualIntegration.ID {
				found = true
				foundIntegration = actualIntegration
				break
			}
		}
		require.True(t, found, "Unable to find integration: %v", expectedIntegration)
		requireDeepEqual(t, expectedIntegration, foundIntegration)
	}
}

// requireDeepEqualArtifactResults asserts that the expected and actual lists of ArtifactResults
// containt the same elements.
func requireDeepEqualArtifactResults(t *testing.T, expected, actual []models.ArtifactResult) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedArtifactResult := range expected {
		found := false
		var foundArtifactResult models.ArtifactResult

		for _, actualArtifactResult := range actual {
			if expectedArtifactResult.ID == actualArtifactResult.ID {
				found = true
				foundArtifactResult = actualArtifactResult
				break
			}
		}
		require.True(t, found, "Unable to find ArtifactResult: %v", expectedArtifactResult)
		requireDeepEqual(t, expectedArtifactResult, foundArtifactResult)
	}
}

// requireDeepEqualDAGs asserts that the expected and actual lists of DAGs
// contain the same elements.
func requireDeepEqualDAGs(t *testing.T, expected, actual []models.DAG) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedDAG := range expected {
		found := false
		var foundDAG models.DAG

		for _, actualDAG := range actual {
			if expectedDAG.ID == actualDAG.ID {
				found = true
				foundDAG = actualDAG
				break
			}
		}
		require.True(t, found, "Unable to find DAG: %v", expectedDAG)
		requireDeepEqual(t, expectedDAG, foundDAG)
	}
}

// requireDeepEqualArtifact asserts that the expected and actual lists of Artifacts
// contain the same elements.
func requireDeepEqualArtifacts(t *testing.T, expected, actual []models.Artifact) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedArtifact := range expected {
		found := false
		var foundArtifact models.Artifact

		for _, actualArtifact := range actual {
			if expectedArtifact.ID == actualArtifact.ID {
				found = true
				foundArtifact = actualArtifact
				break
			}
		}
		require.True(t, found, "Unable to find Artifact: %v", expectedArtifact)
		requireDeepEqual(t, expectedArtifact, foundArtifact)
	}
}

// requireDeepEqualDAGResults asserts that the expected and actual lists
// of DAGResults contain the same elements.
func requireDeepEqualDAGResults(t *testing.T, expected, actual []models.DAGResult) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedDAGResult := range expected {
		found := false
		var foundDAGResult models.DAGResult

		for _, actualDAGResult := range actual {
			if expectedDAGResult.ID == actualDAGResult.ID {
				found = true
				foundDAGResult = actualDAGResult
				break
			}
		}

		require.True(t, found, "Unable to find DAGResult: %v", expectedDAGResult)
		requireDeepEqual(t, expectedDAGResult, foundDAGResult)
	}
}

// requireDeepEqualDAGEdges asserts that the expected and actual lists
// of DAGEdges contain the same elements.
func requireDeepEqualDAGEdges(t *testing.T, expected, actual []models.DAGEdge) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedDAGEdge := range expected {
		found := false
		var foundDAGEdge models.DAGEdge

		for _, actualDAGEdge := range actual {
			if reflect.DeepEqual(expectedDAGEdge, actualDAGEdge) {
				found = true
				foundDAGEdge = actualDAGEdge
				break
			}
		}

		require.True(t, found, "Unable to find DAGEdge: %v", expectedDAGEdge)
		requireDeepEqual(t, expectedDAGEdge, foundDAGEdge)
	}
}

// requireDeepEqualOperators asserts that the expected and actual lists of
// Operators contain the same elements.
func requireDeepEqualOperators(t *testing.T, expected, actual []models.Operator) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedOperator := range expected {
		found := false
		var foundOperator models.Operator

		for _, actualOperator := range actual {
			if expectedOperator.ID == actualOperator.ID {
				found = true
				foundOperator = actualOperator
				break
			}
		}
		require.True(t, found, "Unable to find Operator: %v", expectedOperator)
		requireDeepEqual(t, expectedOperator, foundOperator)
	}
}

// requireDeepEqualExecutionEnvironment asserts that the expected and actual lists of
// ExecutionEnvironment contain the same elements.
func requireDeepEqualExecutionEnvironment(t *testing.T, expected, actual []models.ExecutionEnvironment) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedExecutionEnvironment := range expected {
		found := false
		var foundExecutionEnvironment models.ExecutionEnvironment

		for _, actualExecutionEnvironment := range actual {
			if expectedExecutionEnvironment.ID == actualExecutionEnvironment.ID {
				found = true
				foundExecutionEnvironment = actualExecutionEnvironment
				break
			}
		}
		require.True(t, found, "Unable to find ExecutionEnvironment: %v", expectedExecutionEnvironment)
		requireDeepEqual(t, expectedExecutionEnvironment, foundExecutionEnvironment)
	}
}

// requireDeepEqualLoadOperators asserts that the expected and actual lists of
// LoadOperators contain the same elements.
func requireDeepEqualLoadOperators(t *testing.T, expected, actual []views.LoadOperator) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedOperator := range expected {
		found := false

		for _, actualOperator := range actual {
			if reflect.DeepEqual(expectedOperator, actualOperator) {
				found = true
				break
			}
		}
		require.True(t, found, "Unable to find LoadOperator: %v", expectedOperator)
	}
}

// requireDeepEqualOperatorResults asserts that the expected and actual lists of
// OperatorResults contain the same elements.
func requireDeepEqualOperatorResults(t *testing.T, expected, actual []models.OperatorResult) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedOperatorResult := range expected {
		found := false
		var foundOperatorResult models.OperatorResult

		for _, actualOperatorResult := range actual {
			if expectedOperatorResult.ID == actualOperatorResult.ID {
				found = true
				foundOperatorResult = actualOperatorResult
				break
			}
		}
		require.True(t, found, "Unable to find OperatorResult: %v", expectedOperatorResult)
		requireDeepEqual(t, expectedOperatorResult, foundOperatorResult)
	}
}

// requireDeepEqualOperatorResultStatuses asserts that the expected and actual lists of
// OperatorResultStatuses contain the same elements.
func requireDeepEqualOperatorResultStatuses(t *testing.T, expected, actual []views.OperatorResultStatus) {
	require.Equal(t, len(expected), len(actual))

	for _, expectedOperatorResultStatus := range expected {
		found := false
		var foundOperatorResultStatus views.OperatorResultStatus

		for _, actualOperatorResultStatus := range actual {
			if expectedOperatorResultStatus.ArtifactID == actualOperatorResultStatus.ArtifactID &&
				expectedOperatorResultStatus.DAGResultID == actualOperatorResultStatus.DAGResultID &&
				expectedOperatorResultStatus.OperatorName == actualOperatorResultStatus.OperatorName {

				found = true
				foundOperatorResultStatus = actualOperatorResultStatus
				// Metadata's timestamps is set equal since the timestamps will not match due to the fact
				// that they are pointers.
				expectedOperatorResultStatus.Metadata.Timestamps.PendingAt = actualOperatorResultStatus.Metadata.Timestamps.PendingAt
				break
			}
		}
		require.True(t, found, "Unable to find OperatorResultStatus: %v\nExpected: %v\n Actual: %v", expectedOperatorResultStatus, expected, actual)
		requireDeepEqual(t, expectedOperatorResultStatus, foundOperatorResultStatus)
	}
}
