# Test Case to validate the cspc pool creation is failed when the blockdevice has filesystem already on it.

## Description
   - This test is capable of validating if the pool creation is failed when the specified blockdevices has filesystem.
   - This test constitutes the below files.
     - test_vars.yml - This test_vars file has the list of test specific variables used in this test case.
     - test.yml - Playbook where the test logic is to validate the blockdevice reusability.
     - create_filesystem.yml - Util where the test logic is implemented to create or remove the filesystem. 
     - cspc.yml  - CSPC pool spec which has to be populated with the given variables.
   - This test case should be provided with the parameters in form of job environmental variables in run_e2e_test.yml.

## Environment Variables

| Parameters              | Description                                                                       |
| ----------------------- | --------------------------------------------------------------------------------- |
| OPERATOR_NS             | Namespace where the openebs is deployed                                           |
| POOL_NAME               | Name of the cspc pool                                                             |
| POOL_TYPE               | Type of the pool to create, supported values are striped, mirrored, raidz ,raidz2 |
| NODE_PASSWORD           | password for the node to ssh                                                      |
