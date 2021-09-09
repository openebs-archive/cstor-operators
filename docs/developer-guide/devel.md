# Development Workflow

## Prerequisites

* You have Go 1.13+ installed on your local host/development machine.
* You have Docker installed on your local host/development machine. Docker is required for building cstor-operators component container images and to push them into a Kubernetes cluster for testing.

## Initial Setup

### Fork in the cloud

1. Visit https://github.com/openebs/cstor-operators.
2. Click `Fork` button (top right) to establish a cloud-based fork.

### Clone fork to local host

Place openebs/cstor-operators' code on your local machine using the following cloning procedure.
Create your clone:

Move to the directory where you want to place the clone.

```sh
# Note: Here $user is your GitHub profile name
git clone https://github.com/$user/cstor-operators.git

# Configure remote upstream
cd cstor-operators
git remote add upstream https://github.com/openebs/cstor-operators.git

# Never push to upstream develop
git remote set-url --push upstream no_push

# Confirm that your remotes make sense
git remote -v
```

## Development

### Always sync your local repository

Open a terminal on your local host. Change directory to the cstor-operators-fork root.

```sh
$ cd cstor-operators
```

 Checkout the develop branch.

 ```sh
 $ git checkout develop
 Switched to branch 'develop'
 Your branch is up-to-date with 'origin/develop'.
 ```

 Recall that origin/develop is a branch on your remote GitHub repository.
 Make sure you have the upstream remote openebs/cstor-operators by listing them.

 ```sh
 $ git remote -v
 origin	https://github.com/$user/cstor-operators.git (fetch)
 origin	https://github.com/$user/cstor-operators.git (push)
 upstream	https://github.com/openebs/cstor-operators.git (fetch)
 upstream	no_push (push)
 ```

 If the upstream is missing, add it by using below command.

 ```sh
 $ git remote add upstream https://github.com/openebs/cstor-operators.git
 ```

 Fetch all the changes from the upstream develop branch.

 ```sh
 $ git fetch upstream develop
 remote: Counting objects: 141, done.
 remote: Compressing objects: 100% (29/29), done.
 remote: Total 141 (delta 52), reused 46 (delta 46), pack-reused 66
 Receiving objects: 100% (141/141), 112.43 KiB | 0 bytes/s, done.
 Resolving deltas: 100% (79/79), done.
 From github.com:openebs/cstor-operators
   * branch            develop     -> FETCH_HEAD
 ```

 Rebase your local develop with the upstream/develop.

 ```sh
 $ git rebase upstream/develop
 First, rewinding head to replay your work on top of it...
 Fast-forwarded develop to upstream/develop.
 ```

 This command applies all the commits from the upstream develop to your local develop.

 Check the status of your local branch.

 ```sh
 $ git status
 On branch develop
 Your branch is ahead of 'origin/develop' by 38 commits.
 (use "git push" to publish your local commits)
 nothing to commit, working directory clean
 ```

 Your local repository now has all the changes from the upstream remote. You need to push the changes to your own remote fork which is origin develop.

 Push the rebased develop to origin develop.

 ```sh
 $ git push origin develop
 Username for 'https://github.com': $user
 Password for 'https://$user@github.com':
 Counting objects: 223, done.
 Compressing objects: 100% (38/38), done.
 Writing objects: 100% (69/69), 8.76 KiB | 0 bytes/s, done.
 Total 69 (delta 53), reused 47 (delta 31)
 To https://github.com/$user/cstor-operators.git
 8e107a9..5035fa1  develop -> develop
 ```

### Create a new feature branch to work on your issue

 Your branch name should have the format XX-descriptive where XX is the issue number you are working on followed by some descriptive text. For example:

 ```sh
 $ git checkout -b 1234-fix-developer-docs
 Switched to a new branch '1234-fix-developer-docs'
 ```


### Test your changes
After you are done with your changes, run the unit tests.
 ```sh
 cd cstor-operators

 # Run every unit test
 make test
 ```

### Build the images
You can build amd64 images by following command.
 ```sh
 cd cstor-operators
 make all.amd64
 ```


### Keep your branch in sync

[Rebasing](https://git-scm.com/docs/git-rebase) is very important to keep your branch in sync with the changes being made by others and to avoid huge merge conflicts while raising your Pull Requests. You will always have to rebase before raising the PR.

```sh
# While on your myfeature branch (see above)
git fetch upstream
git rebase upstream/develop
```

While you rebase your changes, you must resolve any conflicts that might arise and build and test your changes using the above steps.

## Submission

### Create a pull request

Before you raise the Pull Requests, ensure you have reviewed the checklist in the [CONTRIBUTING GUIDE](./start.md):
- Ensure that you have re-based your changes with the upstream using the steps above.
- Ensure that you have added the required unit tests for the bug fixes or new feature that you have introduced.
- Ensure your commits history is clean with proper header and descriptions.

Go to the [openebs/cstor-operators github](https://github.com/openebs/cstor-operators) and follow the Open Pull Request link to raise your PR from your development branch.
