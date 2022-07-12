# Contributing

This [link](docs/developer-guide/start.md) contains all information about
building cstor-operators from source, contribution guide, reaching out to
contributors, maintainers etc.

If you want to build the cstor-operators right away then the following is the step:

#### You have a working [Go environment] and [Docker environment].

```
mkdir -p $GOPATH/src/github.com
cd $GOPATH/src/github.com
git clone https://github.com/openebs/cstor-operators openebs/cstor-operators
cd openebs/cstor-operators
make all
```

Alternatively, you can open this repo in Gitpod and make your PR right from the browser:

[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/openebs/cstor-operators)

---

### Sign your work

We use the Developer Certificate of Origin (DCO) as an additional safeguard for the OpenEBS project. This is a well established and widely used mechanism to assure contributors have confirmed their right to license their contribution under the project's license. Please read [developer-certificate-of-origin](./contribute/developer-certificate-of-origin).

Please certify it by just adding a line to every git commit message. Any PR with Commits which does not have DCO Signoff will not be accepted:

```
  Signed-off-by: Random J Developer <random@developer.example.org>
```

or use the command `git commit -s -m "commit message comes here"` to sign-off on your commits.

Use your real name (sorry, no pseudonyms or anonymous contributions). If you set your `user.name` and `user.email` git configs, you can sign your commit automatically with `git commit -s`. You can also use git [aliases](https://git-scm.com/book/en/v2/Git-Basics-Git-Aliases) like `git config --global alias.ci 'commit -s'`. Now you can commit with `git ci` and the commit will be signed.

---
