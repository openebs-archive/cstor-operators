Behavior Driven Development(BDD)

Maya makes use of ginkgo & gomega libraries to implement its integration tests.

To run the BDD test:
1) Copy kubeconfig file into path ~/.kube/config or set the KUBECONFIG env.
2) Change the current directory path to test related directory

Example:

To trigger the cspc sanity test

Step1: Go to the root directory of the cstor-operators repo.
       The current directory should be "*/github.com/openebs/cstor-operator/tests/" 
       in this case
       
Step2: Execute the command `ginkgo -v -r tests/ -- -kubeconfig=/path/to/kubeconfig`

Output:
Sample example output
```
Running Suite: CSPC Sanity Tests
================================
Random Seed: 1586813951
Will run 8 of 8 specs

CSPC Stripe On One Node Provisioning and cleanup of CSPC Creating a cspc 
  no error should be returned
  /home/ashutosh/go/src/github.com/openebs/cstor-operators/tests/cspc/sanity/cspc_sanity_test.go:68
•
------------------------------
CSPC Stripe On One Node Provisioning and cleanup of CSPC Creating a cspc 
  desired count should be 1 on cspc
  /home/ashutosh/go/src/github.com/openebs/cstor-operators/tests/cspc/sanity/cspc_sanity_test.go:74

```
Note: Above is the sample output how it looks when you ran `ginkgo -v -r tests/ -- -kubeconfig=/path/to/kubeconfig`