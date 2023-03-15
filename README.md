# Stack Auditor

![Stack Auditor Logo](logo.png "Stack Auditor Logo")

## Installation

* Download the latest stack-auditor from the [release section](https://github.com/cloudfoundry/stack-auditor/releases) of this repository. 
* Unpack the archive on your local box using `tar xvzf <archive> [-C <directory>]` or use a file explored. 
* Install the plugin with `cf install-plugin <path_to_binary>`.

### Alternative: Compile from source

Prerequisite: Have a working golang environment with correctly set
`GOPATH`.

```sh
go get github.com/cloudfoundry/stack-auditor
cd $GOPATH/src/github.com/cloudfoundry/stack-auditor
./scripts/build.sh

```

## Usage

Install the plugin with `cf install-plugin <path_to_binary>` or use the shell scripts `./scripts/install.sh` or `./scripts/reinstall.sh`.

* Audit cf applications using `cf audit-stack [--csv | --json]`. These optional flags return csv or json format instead of plain text.
* Change stack association using `cf change-stack <app> <stack>`. This will attempt to perform a zero downtime restart. Make sure to target the space that contains the app you want to re-associate. 
* Delete a stack using `cf delete-stack <stack> [--force | -f]`

## Run the Tests

Target a cloudfoundry with the following prerequisites:
  - has cflinuxfs3 and cflinuxfs4 stacks and buildpacks
    - If using cf-deployment, this can be enabled with the ops file `operations/experimental/add-cflinuxfs4.yml`
  - you are targeting an org and a space

Then run:

`./scripts/all-tests.sh`
