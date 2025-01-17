import pytest
from aqueduct.error import AqueductError

from aqueduct import op

from ..shared.flow_helpers import publish_flow_test


def test_multiple_outputs(client, flow_name, engine):
    @op(num_outputs=2)
    def generate_two_outputs():
        return "hello", 1234

    @op
    def append_to_str(input_str):
        return input_str + " world."

    @op
    def double_number(num):
        return 2 * num

    str_artifact, int_artifact = generate_two_outputs()
    assert str_artifact.get() == "hello"
    assert int_artifact.get() == 1234
    assert str_artifact.name() == "generate_two_outputs artifact"
    assert int_artifact.name() == "generate_two_outputs artifact"

    str_output = append_to_str(str_artifact)
    int_output = double_number(int_artifact)
    assert str_output.get() == "hello world."
    assert int_output.get() == 2468

    flow = publish_flow_test(
        client,
        name=flow_name(),
        artifacts=[str_output, int_output],
        engine=engine,
    )

    # Check the default naming of multi-output functions.
    flow_run = flow.latest()
    assert flow_run.artifact("generate_two_outputs artifact").get() == "hello"
    assert flow_run.artifact("generate_two_outputs artifact (1)").get() == 1234


def test_multiple_outputs_user_failure(client):
    @op(num_outputs=3)
    def generate_two_outputs():
        return "hello", 1234

    with pytest.raises(AqueductError):
        output_artifact = generate_two_outputs()
        output_artifact.get()
