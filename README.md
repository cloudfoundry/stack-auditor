# Stack Auditor

## Usage
Download the latest binary for your operating system from the [release section](https://github.com/cloudfoundry/stack-auditor/releases) of this repository. Install the plugin with `cf install-plugin <path_to_binary>`.

Audit cf applications using `cf audit-stack` and change stack association using `cf change-stack <app> <stack>`.

## Run the Tests

Target a cloudfoundry and run:

`./scripts/all-tests.sh` 
