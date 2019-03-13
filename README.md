# Stack Auditor

## Installation
* Download the latest stack-auditor from the [release section](https://github.com/cloudfoundry/stack-auditor/releases) of this repository. 
* Unpack the archive on your local box using `tar xvzf <archive> [-C <directory>]` or use a file explored. 
* Install the plugin with `cf install-plugin <path_to_binary>`.

## Usage
Audit cf applications using `cf audit-stack` and change stack association using `cf change-stack <app> <stack>`.

## Run the Tests

Target a cloudfoundry and run:

`./scripts/all-tests.sh` 
