from aqueduct.constants.enums import LoadUpdateMode

from ..shared.data_objects import DataObject
from ..shared.utils import extract, generate_table_name, publish_flow_test
from .save import save


def test_multiple_artifacts_saved_to_same_integration(
    client, flow_name, data_integration, engine, validator
):
    table_1_save_name = generate_table_name()
    table_2_save_name = generate_table_name()

    table_1 = extract(data_integration, DataObject.SENTIMENT)
    save(data_integration, table_1, name=table_1_save_name, update_mode=LoadUpdateMode.REPLACE)
    table_2 = extract(data_integration, DataObject.SENTIMENT)
    save(data_integration, table_2, name=table_2_save_name, update_mode=LoadUpdateMode.REPLACE)

    flow = publish_flow_test(
        client,
        name=flow_name(),
        artifacts=[table_1, table_2],
        engine=engine,
    )

    validator.check_saved_artifact_data(flow, table_1.id(), expected_data=table_1.get())
    validator.check_saved_artifact_data(flow, table_2.id(), expected_data=table_2.get())