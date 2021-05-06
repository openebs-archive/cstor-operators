# Test Case to validate the reusability of blockdevice.

## Description
   - This test is capable of validate the blockdevice reusability. 
   - First it will get the list of blockdevices used for existing SPC pool or the CSPC pool
   - And then delete the SPC pool or CSPC pool then the same blockdevices obtained from existing SPC and CSPC pool used for creating the new cspc pool.
   - This test constitutes the below files.
     - test_vars.yml - This test_vars file has the list of test specific variables used in this test case.
     - test.yml - Playbook where the test logic is to validate the blockdevice reusability.
     - blockdevice.j2 - Template to append the identified blockdevice to create the pool.
     - add_blockdevice.yml - Util where the test logic is implemented to obtain the blockdevice list. 
     - cspc.yml  - CSPC pool spec which has to be populated with the given variables.
   - This test case should be provided with the parameters in form of job environmental variables in run_e2e_test.yml.

## Environment Variables

| Parameters              | Description                                                                       |
| ----------------------- | --------------------------------------------------------------------------------- |
| OPERATOR_NS             | Namespace where the openebs is deployed                                           |
| SPC_POOL_NAME           | Name of the spc pool                                                              |
| CSPC_POOL_NAME          | Name of the cspc Pool                                                             |
| POOL_TYPE               | Type of the pool to create, supported values are striped, mirror, raidz ,raidz2   |
| STORAGE_CLASS           | Name of the storage class to deploy the application                               |
| ACTION                  | Value to provision or deprovision the pool                                        |
