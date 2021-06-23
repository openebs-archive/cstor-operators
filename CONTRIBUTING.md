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
