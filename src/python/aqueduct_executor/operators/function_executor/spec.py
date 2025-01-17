import json
from typing import Any, List, Optional

try:
    from typing import Literal
except ImportError:
    # Python 3.7 does not support typing.Literal
    from typing_extensions import Literal  # type: ignore

from aqueduct_executor.operators.utils.enums import (
    ArtifactType,
    CheckSeverity,
    JobType,
    OperatorType,
)
from aqueduct_executor.operators.utils.storage import config
from pydantic import BaseModel, Extra, parse_obj_as


class FunctionSpec(BaseModel):
    name: str
    type: Literal[JobType.FUNCTION]
    storage_config: config.StorageConfig
    metadata_path: str
    function_path: str
    function_extract_path: str
    entry_point_file: str
    entry_point_class: str
    entry_point_method: str
    custom_args: str
    input_content_paths: List[str]
    input_metadata_paths: List[str]
    output_content_paths: List[str]
    output_metadata_paths: List[str]
    expected_output_artifact_types: List[ArtifactType]
    operator_type: OperatorType

    # This is specific to the check operator. This is left unset by any other function type.
    check_severity: Optional[CheckSeverity]

    # This is always unset - it is only here because we forbid extra fields.
    resources: Optional[Any]

    class Config:
        extra = Extra.forbid


def parse_spec(spec_json: bytes) -> FunctionSpec:
    """
    Parses a JSON string into a FunctionSpec.
    """
    data = json.loads(spec_json)
    return parse_obj_as(FunctionSpec, data)
