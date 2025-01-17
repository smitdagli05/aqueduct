# SDK Integration Tests

These tests run the SDK against an Aqueduct backend. Each test is built to clean up after itself. If it creates a workflow, it will attempt to delete it in the end. All tests can be run in parallel.

There are two different suites of SDK Integration Tests, each with their own purpose:
1) Aqueduct Tests: These tests cover all Aqueduct behavior, from a user's perspective. They set up DAGs and usually publish flows. 
Test cases are generic enough to be reusable across multiple types of data integrations and engines. They are found in the `aqueduct_tests/` folder.
2) Data Integration Tests: While Aqueduct tests is the defacto test suite for testing Aqueduct behavior, one disadvantage of such
powerful but generic test cases is that every data integration is different. Unlike compute, each data integration has its own set of
APIs, abilities, and limitations, and Aqueduct Tests are philosophically less suitable for providing such coverage. Data Integration tests are meant
to be focused and complete, instead of reusable. They should only use the SDK's Integration API to validate data movement to and from
our supported third-party integrations.

## Configuration
For these test suites to run, a configuration file must exist at `test-config.yml`. See `test-config-example.yml` for the format template.
This file contains:
1) The apikey to access the server.
2) The server's address.
3) The connection configuration information for each of the data integrations to run against. The test suites
will automatically run against each of the data integrations specified in this file, unless a `--data` argument
is supplied, whereby the tests will filter down to just that data integration.

Both these test suites share a collection of custom command line flags:
* `--data`: The integration name of the data integration to run all tests against. 
* `--engine`: The integration of the engine to compute all tests on.
* `--keep-flows`: If set, we will not delete any flows created by the test run. This is useful for debugging.
* `--deprecated`: Runs against any deprecated API that still exists in the SDK. Such code paths should be eventually deleted after some time, but this ensures backwards compatibility.
* `--skip-data-setup`: Skips the checking and setup of external data integrations. Instead, assumes that all data integrations have been set up correctly with the appropriate data.
* `--skip-engine-setup`: Skips the checking and setup of external compute integrations.

For additional markers/fixtures/flags, please inspect `conftest.py` in this directory. For test-specific configurations,
see `aqueduct_tests/conftest.py` and  `data_integration_tests/conftest.py`.

## Running the Tests
You can run this test suite using vanilla pytest, or you can use the `run_tests.py` script in this directory.

### About run_tests.py
`run_tests.py` is just a convenience wrapper around the pytest command. Any pytest flags can be used
with this script too. The main difference is that run_tests.py adds some default pytest configuration,
like setting the default concurrenty to 8.

### Commands
Note that to run tests with concurrency > 1, `pytest-xdist` must be installed.

To run all SDK Integration Tests, from the `integration_tests/sdk` directory, run:
`python3 run_tests.py`

To run just one of the test suites:
- `python3 run_tests.py --aqueduct`
- `python3 run_tests.py --data-integration`

To run just one of the test files:
- `python3 run_tests.py --aqueduct --file flow_test.py`

To run just one test case:
- `python3 run_tests.py --aqueduct -k test_basic_flow`

### Useful Command Flags 
In addition to the custom test suite flags listed above, you can also apply generic pytest flags to the test run too.

For example, to only run tests that have failed in the last run, use the `--lf` flag.
- `python3 run_tests.py --lf`