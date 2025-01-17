import importlib
import json
import os
import shutil
import sys
import tracemalloc
import uuid
from typing import Any, Callable, Dict, List, Tuple

import numpy as np
import pandas as pd
from aqueduct.utils.type_inference import infer_artifact_type
from aqueduct_executor.operators.function_executor import extract_function, get_extract_path
from aqueduct_executor.operators.function_executor.execute import (
    cleanup,
    get_py_import_path,
    import_invoke_method,
    validate_spec,
)
from aqueduct_executor.operators.function_executor.spec import FunctionSpec
from aqueduct_executor.operators.function_executor.utils import OP_DIR
from aqueduct_executor.operators.spark.utils import read_artifacts_spark, write_artifact_spark
from aqueduct_executor.operators.utils import utils
from aqueduct_executor.operators.utils.enums import (
    ArtifactType,
    CheckSeverity,
    ExecutionStatus,
    FailureType,
    OperatorType,
    SerializationType,
)
from aqueduct_executor.operators.utils.execution import (
    TIP_CHECK_DID_NOT_PASS,
    TIP_NOT_BOOL,
    TIP_NOT_NUMERIC,
    TIP_OP_EXECUTION,
    TIP_UNKNOWN_ERROR,
    ExecFailureException,
    ExecutionState,
    Logs,
    exception_traceback,
)
from aqueduct_executor.operators.utils.storage.parse import parse_storage
from aqueduct_executor.operators.utils.timer import Timer
from pyspark.sql import SparkSession, dataframe


def _infer_artifact_type_spark(value: Any) -> Any:
    if isinstance(value, dataframe.DataFrame):
        return ArtifactType.TABLE
    else:
        return infer_artifact_type(value)


def _execute_function(
    spec: FunctionSpec,
    inputs: List[Any],
    exec_state: ExecutionState,
) -> Tuple[List[Any], Dict[str, str]]:
    """
    Invokes the given function on the input data. Does not raise an exception on any
    user function errors, but instead annotates the given exec state with the error.

    :param inputs: the input data to feed into the user's function.
    """

    invoke = import_invoke_method(spec)
    timer = Timer()
    timer.start()
    tracemalloc.start()

    @exec_state.user_fn_redirected(failure_tip=TIP_OP_EXECUTION)
    def _invoke() -> Any:
        return invoke(*inputs)

    results = _invoke()
    if len(spec.output_content_paths) == 1:
        results = [results]

    elapsedTime = timer.stop()
    _, peak = tracemalloc.get_traced_memory()
    system_metadata = {
        utils._RUNTIME_SEC_METRIC_NAME: str(elapsedTime),
        utils._MAX_MEMORY_MB_METRIC_NAME: str(peak / 10**6),
    }

    sys.path.pop(0)
    return results, system_metadata


def _validate_result_count_and_infer_type(
    spec: FunctionSpec,
    results: List[Any],
) -> List[ArtifactType]:
    """
    Validates that the expected number of results were returned by the Function
    and infers the ArtifactType of each result.

    Args:
        spec: The FunctionSpec for the Function
        results: The results returned by the Function

    Returns:
        The ArtifactType of each result

    Raises:
        ExecFailureException: If the expected number of results were not returned
    """
    if len(spec.output_content_paths) > 1 and len(spec.output_content_paths) != len(results):
        raise ExecFailureException(
            failure_type=FailureType.USER_FATAL,
            tip="Expected function to have %s outputs, but instead it had %s."
            % (len(spec.output_content_paths), len(results)),
        )

    return [_infer_artifact_type_spark(res) for res in results]


def run(spec: FunctionSpec, spark_session_obj: SparkSession) -> None:
    """
    Executes a function operator.
    """
    print("Started %s job: %s" % (spec.type, spec.name))

    exec_state = ExecutionState(user_logs=Logs())
    storage = parse_storage(spec.storage_config)
    try:
        validate_spec(spec)

        # Read the input data from intermediate storage.
        inputs, _, serialization_types = read_artifacts_spark(
            storage,
            spec.input_content_paths,
            spec.input_metadata_paths,
            spark_session_obj,
        )

        derived_from_bson = SerializationType.BSON_TABLE in serialization_types
        print("Invoking the function...")
        results, system_metadata = _execute_function(spec, inputs, exec_state)
        if exec_state.status == ExecutionStatus.FAILED:
            # user failure
            utils.write_exec_state(storage, spec.metadata_path, exec_state)
            sys.exit(1)

        print("Function invoked successfully!")

        result_types = _validate_result_count_and_infer_type(spec, results)

        # Perform type checking on the function output.
        if spec.operator_type == OperatorType.METRIC:
            assert len(results) == 1, "Metric operator can only have a single output."
            result = results[0]

            if not (
                isinstance(result, int)
                or isinstance(result, float)
                or isinstance(result, np.number)
            ):
                raise ExecFailureException(
                    failure_type=FailureType.USER_FATAL,
                    tip=TIP_NOT_NUMERIC,
                )

        elif spec.operator_type == OperatorType.CHECK:
            assert len(results) == 1, "Check operator can only have a single output."
            check_result = results[0]

            if isinstance(check_result, pd.Series) and check_result.dtype == "bool":
                assert result_types[0] == ArtifactType.PICKLABLE

                # Cast pd.Series to a bool.
                # We only write True if every boolean in the series is True.
                series = pd.Series(check_result)
                check_passed = bool(series.size - series.sum().item() == 0)
            elif isinstance(check_result, bool) or isinstance(check_result, np.bool_):
                # Cast np.bool_ to a bool.
                check_passed = bool(check_result)
            else:
                raise ExecFailureException(
                    failure_type=FailureType.USER_FATAL,
                    tip=TIP_NOT_BOOL,
                )

            # If the check returned a value we interpret to mean 'false', we exit here, but
            # not before recording the output artifact value (which will be False).
            if not check_passed:
                print(f"Check Operator did not pass.")

                write_artifact_spark(
                    storage,
                    ArtifactType.BOOL,
                    derived_from_bson,  # derived_from_bson doesn't apply to bool artifact
                    spec.output_content_paths[0],
                    spec.output_metadata_paths[0],
                    check_passed,
                    system_metadata=system_metadata,
                    spark_session_obj=spark_session_obj,
                )

                check_severity = spec.check_severity
                if spec.check_severity is None:
                    print(
                        "Check operator has an unspecified severity on spec. Defaulting to ERROR."
                    )
                    check_severity = CheckSeverity.ERROR

                failure_type = FailureType.USER_FATAL
                if check_severity == CheckSeverity.WARNING:
                    failure_type = FailureType.USER_NON_FATAL

                raise ExecFailureException(failure_type, tip=TIP_CHECK_DID_NOT_PASS)

            # If we get here, we know that the check has passed. The artifact type might need
            # still be updated. Eg. if the output was a pandas series.
            result_types[0] = ArtifactType.BOOL
            results[0] = True
        else:
            for i, expected_output_type in enumerate(spec.expected_output_artifact_types):
                if (
                    expected_output_type != ArtifactType.UNTYPED
                    and expected_output_type != result_types[i]
                ):
                    raise ExecFailureException(
                        failure_type=FailureType.USER_FATAL,
                        tip="Expected type %s for the %d-th output of function, but it is of type %s."
                        % (expected_output_type, i, result_types[i]),
                    )

        for i, result in enumerate(results):
            write_artifact_spark(
                storage,
                result_types[i],
                derived_from_bson,
                spec.output_content_paths[i],
                spec.output_metadata_paths[i],
                result,
                system_metadata=system_metadata,
                spark_session_obj=spark_session_obj,
            )

        # If we made it here, then the operator has succeeded.
        exec_state.status = ExecutionStatus.SUCCEEDED
        print(f"Succeeded! Full logs: {exec_state.json()}")
        utils.write_exec_state(storage, spec.metadata_path, exec_state)

    except ExecFailureException as e:
        # We must reconcile the user logs here, since those logs are not captured on the exception.
        from_exception_exec_state = ExecutionState.from_exception(e, user_logs=exec_state.user_logs)
        print(f"Failed with error. Full Logs:\n{from_exception_exec_state.json()}")
        utils.write_exec_state(storage, spec.metadata_path, from_exception_exec_state)
        sys.exit(1)

    except Exception as e:
        exec_state.mark_as_failure(
            FailureType.SYSTEM, TIP_UNKNOWN_ERROR, context=exception_traceback(e)
        )
        print(f"Failed with system error. Full Logs:\n{exec_state.json()}")
        utils.write_exec_state(storage, spec.metadata_path, exec_state)
        sys.exit(1)
    finally:
        # Perform any cleanup
        cleanup(spec)
