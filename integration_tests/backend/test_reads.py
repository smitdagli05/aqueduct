import json
import os
import uuid
from pathlib import Path

import pytest
import requests
import utils
from exec_state import assert_exec_state
from setup.changing_saves_workflow import setup_changing_saves
from setup.flow_with_failure import setup_flow_with_failure
from setup.flow_with_metrics_and_checks import setup_flow_with_metrics_and_checks
from setup.flow_with_sleep import setup_flow_with_sleep

import aqueduct
from aqueduct import globals


class TestBackend:
    GET_WORKFLOWS_TEMPLATE = "/api/v2/workflows"

    LIST_WORKFLOW_SAVED_OBJECTS_TEMPLATE = "/api/workflow/%s/objects"
    GET_TEST_INTEGRATION_TEMPLATE = "/api/integration/%s/test"
    LIST_INTEGRATIONS_TEMPLATE = "/api/integrations"
    CONNECT_INTEGRATION_TEMPLATE = "/api/integration/connect"
    DELETE_INTEGRATION_TEMPLATE = "/api/integration/%s/delete"
    GET_WORKFLOW_RESULT_TEMPLATE = "/api/workflow/%s/result/%s"
    LIST_ARTIFACT_RESULTS_TEMPLATE = "/api/workflow/%s/artifact/%s/results"

    WORKFLOW_PATH = Path(__file__).parent / "setup"
    DEMO_DB_PATH = os.path.join(os.environ["HOME"], ".aqueduct/server/db/demo.db")

    @classmethod
    def setup_class(cls):
        cls.client = aqueduct.Client(pytest.api_key, pytest.server_address)
        cls.integration = cls.client.integration(name=pytest.integration)
        cls.flows = {
            "changing_saves": setup_changing_saves(cls.client, pytest.integration),
            "flow_with_failure": setup_flow_with_failure(cls.client, pytest.integration),
            "flow_with_metrics_and_checks": setup_flow_with_metrics_and_checks(
                cls.client,
                pytest.integration,
            ),
            # this flow is intended to provide 'noise' of op / artf with the same name,
            # but under different flow.
            "another_flow_with_metrics_and_checks": setup_flow_with_metrics_and_checks(
                cls.client,
                pytest.integration,
                workflow_name="another_flow_with_metrics_and_checks",
            ),
        }

        # we do not call `wait_for_flow_runs` on these flows
        cls.running_flows = {
            "flow_with_sleep": setup_flow_with_sleep(cls.client, pytest.integration),
        }
        for flow_id, n_runs in cls.flows.values():
            utils.wait_for_flow_runs(cls.client, flow_id, n_runs)

    @classmethod
    def teardown_class(cls):
        for flow_id, _ in cls.flows.values():
            utils.delete_flow(cls.client, flow_id)

        for flow_id, _ in cls.running_flows.values():
            utils.delete_flow(cls.client, flow_id)

    @classmethod
    def response(cls, endpoint, additional_headers):
        headers = {"api-key": pytest.api_key}
        headers.update(additional_headers)
        url = globals.__GLOBAL_API_CLIENT__.construct_full_url(endpoint)
        return url, headers

    @classmethod
    def get_response(cls, endpoint, additional_headers={}):
        url, headers = cls.response(endpoint, additional_headers)
        r = requests.get(url, headers=headers)
        return r

    @classmethod
    def post_response(cls, endpoint, additional_headers={}):
        url, headers = cls.response(endpoint, additional_headers)
        r = requests.post(url, headers=headers)
        return r

    def test_endpoint_list_workflow_tables(self):
        endpoint = self.LIST_WORKFLOW_SAVED_OBJECTS_TEMPLATE % self.flows["changing_saves"][0]
        data = self.get_response(endpoint).json()["object_details"]

        assert len(data) == 3

        # table_name, update_mode
        data_set = set(
            [
                ("table_1", "append"),
                ("table_1", "replace"),
                ("table_2", "replace"),
            ]
        )

        print(data)
        assert (
            set(
                [
                    (item["spec"]["parameters"]["table"], item["spec"]["parameters"]["update_mode"])
                    for item in data
                ]
            )
            == data_set
        )

        # Check all in same integration
        assert len(set([item["integration_name"] for item in data])) == 1
        assert len(set([item["spec"]["service"] for item in data])) == 1

    def test_endpoint_delete_integration(self):
        integration_name = f"test_delete_integration_{uuid.uuid4().hex[:8]}"

        # Check integration did not exist
        data = self.get_response(self.LIST_INTEGRATIONS_TEMPLATE).json()
        assert integration_name not in set([integration["name"] for integration in data])

        # Create integration
        status = self.post_response(
            self.CONNECT_INTEGRATION_TEMPLATE,
            additional_headers={
                "integration-name": integration_name,
                "integration-service": "SQLite",
                "integration-config": json.dumps({"database": self.DEMO_DB_PATH}),
            },
        ).status_code
        assert status == 200

        # Check integration created
        data = self.get_response(self.LIST_INTEGRATIONS_TEMPLATE).json()
        integration_data = {integration["name"]: integration["id"] for integration in data}
        assert integration_name in set(integration_data.keys())

        # Delete integration
        status = self.post_response(
            self.DELETE_INTEGRATION_TEMPLATE % integration_data[integration_name]
        ).status_code
        assert status == 200

        # Check integration does not exist
        data = self.get_response(self.LIST_INTEGRATIONS_TEMPLATE).json()
        assert integration_name not in set([integration["name"] for integration in data])

    def test_endpoint_test_integration(self):
        resp = self.get_response(self.GET_TEST_INTEGRATION_TEMPLATE % self.integration.id())
        assert resp.ok

    def test_endpoint_get_workflow_dag_result_with_failure(self):
        flow_id = self.flows["flow_with_failure"][0]
        flow = self.client.flow(flow_id)
        runs = flow.list_runs()
        resp = self.get_response(
            self.GET_WORKFLOW_RESULT_TEMPLATE % (flow_id, runs[0]["run_id"])
        ).json()
        assert_exec_state(resp["result"]["exec_state"], "failed")
        # operators
        operators = resp["operators"]
        assert len(operators) == 3
        for op in operators.values():
            name = op["name"]
            exec_state = op["result"]["exec_state"]

            if "query" in name:  # extract
                assert_exec_state(exec_state, "succeeded")
            elif name == "bad_op":
                assert_exec_state(exec_state, "failed")
            elif name == "bad_op_downstream":
                assert_exec_state(exec_state, "canceled")
            else:
                raise Exception(f"unexpected operator name {name}")

        # artifacts
        artifacts = resp["artifacts"]
        assert len(artifacts) == 3
        for artf in artifacts.values():
            name = artf["name"]
            exec_state = artf["result"]["exec_state"]

            if "query" in name:
                assert_exec_state(exec_state, "succeeded")
            elif name == "bad_op artifact":
                assert_exec_state(exec_state, "canceled")
            elif name == "bad_op_downstream artifact":
                assert_exec_state(exec_state, "canceled")
            else:
                raise Exception(f"unexpected operator name {name}")

    def test_endpoint_get_workflow_dag_result_with_metrics_and_checks(self):
        flow_id = self.flows["flow_with_metrics_and_checks"][0]
        flow = self.client.flow(flow_id)
        runs = flow.list_runs()
        resp = self.get_response(
            self.GET_WORKFLOW_RESULT_TEMPLATE % (flow_id, runs[0]["run_id"])
        ).json()
        assert_exec_state(resp["result"]["exec_state"], "succeeded")

        # operators
        operators = resp["operators"]
        assert len(operators) == 3
        for op in operators.values():
            name = op["name"]
            exec_state = op["result"]["exec_state"]
            if "query" in name or name == "size" or name == "check":  # extract
                assert_exec_state(exec_state, "succeeded")
            else:
                raise Exception(f"unexpected operator name {name}")

        # artifacts
        artifacts = resp["artifacts"]
        assert len(artifacts) == 3
        for artf in artifacts.values():
            name = artf["name"]
            exec_state = artf["result"]["exec_state"]
            value = artf["result"]["content_serialized"]

            if "query" in name:
                assert_exec_state(exec_state, "succeeded")
            elif name == "size artifact":
                assert_exec_state(exec_state, "succeeded")
                assert int(value) > 0
            elif name == "check artifact":
                assert_exec_state(exec_state, "succeeded")
                assert value == "true"
            else:
                raise Exception(f"unexpected operator name {name}")

    def test_endpoint_get_workflow_dag_result_on_flow_with_sleep(self):
        flow_id = self.running_flows["flow_with_sleep"][0]
        flow = self.client.flow(flow_id)
        runs = flow.list_runs()
        resp = self.get_response(
            self.GET_WORKFLOW_RESULT_TEMPLATE % (flow_id, runs[0]["run_id"])
        ).json()
        assert_exec_state(resp["result"]["exec_state"], "pending")

        # operators
        operators = resp["operators"]
        assert len(operators) == 2
        for op in operators.values():
            name = op["name"]
            exec_state = op["result"]["exec_state"]
            if "query" in name:  # extract
                assert_exec_state(exec_state, "succeeded")
            elif name == "sleeping_op":
                assert_exec_state(exec_state, "pending")
            else:
                raise Exception(f"unexpected operator name {name}")

        # artifacts
        artifacts = resp["artifacts"]
        assert len(artifacts) == 2
        for artf in artifacts.values():
            name = artf["name"]
            exec_state = artf["result"]["exec_state"]

            if "query" in name:
                assert_exec_state(exec_state, "succeeded")
            elif name == "sleeping_op artifact":
                assert_exec_state(exec_state, "pending")
            else:
                raise Exception(f"unexpected operator name {name}")

    def test_endpoint_list_artifact_results_with_metrics_and_checks(self):
        flow_id, num_runs = self.flows["flow_with_metrics_and_checks"]
        flow = self.client.flow(flow_id)
        runs = flow.list_runs()
        resp = self.get_response(
            self.GET_WORKFLOW_RESULT_TEMPLATE % (flow_id, runs[0]["run_id"])
        ).json()

        # artifacts
        artifacts = resp["artifacts"]
        assert len(artifacts) == 3
        for artf in artifacts.values():
            name = artf["name"]
            id = artf["id"]
            resp = self.get_response(self.LIST_ARTIFACT_RESULTS_TEMPLATE % (flow_id, id)).json()
            results = resp["results"]
            assert len(results) == num_runs

            for result in results:
                exec_state = result["exec_state"]
                value = result["content_serialized"]
                assert_exec_state(exec_state, "succeeded")

                if "query" in name:
                    assert value is None
                elif name == "size artifact":
                    assert int(value) > 0
                elif name == "check artifact":
                    assert value == "true"

    def test_endpoint_workflows_get(self):
        resp = self.get_response(self.GET_WORKFLOWS_TEMPLATE)
        resp = resp.json()

        if len(resp) > 0:
            keys = [
                "id",
                "user_id",
                "name",
                "description",
                "schedule",
                "created_at",
                "retention_policy",
                "notification_settings",
            ]

            user_id = resp[0]["user_id"]

            for v2_workflow in resp:
                for key in keys:
                    assert key in v2_workflow
                assert v2_workflow["user_id"] == user_id
