from typing import Union

import pytest
from aqueduct.enums import ExecutionStatus
from aqueduct.error import AqueductError
from utils import run_flow_test, wait_for_flow_runs

from aqueduct import op


@op
def output_different_types(should_return_num: bool) -> Union[str, int]:
    if should_return_num:
        return 123
    return "not a number"


@pytest.mark.publish
def test_flow_fails_on_unexpected_type_output(client):
    type_toggle = client.create_param("output_type_toggle", True)
    output = output_different_types(type_toggle)

    flow = run_flow_test(client, artifacts=[output], delete_flow_after=False)

    try:
        client.trigger(flow.id(), parameters={"output_type_toggle": False})
        wait_for_flow_runs(
            client,
            flow.id(),
            expect_statuses=[ExecutionStatus.SUCCEEDED, ExecutionStatus.FAILED],
        )
    finally:
        client.delete_flow(flow.id())


@pytest.mark.publish
def test_flow_fails_on_unexpected_type_output_for_lazy(client):
    type_toggle = client.create_param("output_type_toggle", True)
    output = output_different_types.lazy(type_toggle)

    # The flow will first be lazily executed, and the new type information
    # will be persisted to the database.
    flow = run_flow_test(client, artifacts=[output], delete_flow_after=False)

    try:
        # Because we are violating our inferred types, this will fail!
        client.trigger(flow.id(), parameters={"output_type_toggle": False})
        wait_for_flow_runs(
            client,
            flow.id(),
            expect_statuses=[ExecutionStatus.SUCCEEDED, ExecutionStatus.FAILED],
        )
    finally:
        client.delete_flow(flow.id())


def test_preview_artifact_backfilled_with_wrong_type(client):
    """An error should be thrown if an upstream operator is previewed with the wrong type."""
    type_toggle = client.create_param("output_type_toggle", True)

    # Sets the type of this artifact eagerly due to preview.
    output = output_different_types(type_toggle)

    @op
    def noop(data):
        return data

    # Lazily execute the downstream operator with a custom parameter.
    # We execute lazily so that the terminal node can expect any output and therefore won't error.
    # We want to induce the upstream type backfill to error.
    noop_output = noop.lazy(output)

    # Fails because the upstream operator should have a type mismatch.
    with pytest.raises(AqueductError, match="Operator `output_different_types` failed!"):
        noop_output.get(parameters={"output_type_toggle": False})