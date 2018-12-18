#!/usr/bin/env bash

# steps

# make sure we actually can cf push to a valid environment?
# make an array of app names
# for every app in this array, push this app
# makeApp prefix count
# Split count into fs2 and fs3

# audit? -> elliot has this

# teardown prefix count
# validate login
# cf delete appname

AppName=testGeneratedApp
AppPath=./fixtures/simple_app
AppBuildpack=nodejs_buildpack


# input $1=number of app instances
makeApps () {
    STACKS=(cflinuxfs2 cflinuxfs3)

    for i in $(seq 1 $1); do
     idx=$(( i % 2 ))
     cf push $AppName-$i -p $AppPath -b $AppBuildpack -s ${STACKS[$idx]} &> /dev/null &
    done
}

deleteApps () {
    set -e

    for i in $(seq 1 $1); do
     cf delete $AppName-$i -f
    done

    set +e
}

