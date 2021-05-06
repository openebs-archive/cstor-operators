# Test Case to validate the cspc pool creation is failed when the blockdevice is already claimed.

## Description
   - This test is capable of validating if the pool creation is failed when the specified blockdevices are already claimed
   - This test constitutes the below files.
     - test_vars.yml - This test_vars file has the list of test specific variables used in this test case.
     - test.yml - Playbook where the test logic is to validate the blockdevice reusability.
     - blockdevice.j2 - Template to append the identified blockdevice to create the pool.
     - add_blockdevice.yml - Util where the test logic is implemented to obtain the claimed blockdevice list. 
     - cspc.yml  - CSPC pool spec which has to be populated with the given variables.
   - This test case should be provided with the parameters in form of job environmental variables in run_e2e_test.yml.

## Environment Variables

| Parameters              | Description                                                                       |
| ----------------------- | --------------------------------------------------------------------------------- |
| OPERATOR_NS             | Namespace where the openebs is deployed                                           |
| POOL_NAME               | Name of the cspc pool                                                              |
| POOL_COUNT              | Number of the pools to be create.                                                 |
| POOL_TYPE               | Type of the pool to create, supported values are striped, mirrored, raidz ,raidz2 |
