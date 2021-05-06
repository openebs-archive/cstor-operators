## Test Case to validate the deletion of cspc pool is blocked if the pool has volumes.

## Description
   - This test case is capable of validating if the pool deletion is blocked when it has volumes.
   - This test constitutes the below files.
     - test_vars.yml - This test_vars file has the list of test specific variables used in E2E-TEST.
     - test.yml - Playbook where the test logic is built to validate the cspc pool deletion.
   - This test case should be provided with the parameters in form of job environmental variables in run_e2e_test.yml

## E2E-TEST Environment Variables

| Parameters              | Description                                                        |
| ----------------------- | ------------------------------------------------------------------ |
| OPERATOR_NS             | Namespace where the openebs is deployed                            |
| APP_NAMESPACE           | Namespace for the application that deployed using cStor cspc pool  |
| APP_LABEL               | Label name of the application                                      |
