#!/usr/bin/env bash

XC_OS=$(go env GOOS)
XC_ARCH=$(go env GOARCH)
DEST_BIN=terraform-provider-datadog

echo "Compiling for OS: $XC_OS and ARCH: $XC_ARCH"

gox -os="${XC_OS}" -arch="${XC_ARCH}"

if [ $? != 0 ] ; then
    echo "Failed to compile, bailing."
    exit 1
fi

echo "Looking for Terraform install"
TERRAFORM_LOC=$(dirname $(which terraform))

if [ $TERRAFORM_LOC  ] ; then
    DEST_PATH=$TERRAFORM_LOC
else
    DEST_PATH=$GOPATH/bin
fi

echo ""
echo "Moving terraform-provider-datadog_${XC_OS}_${XC_ARCH} to $DEST_PATH/$DEST_BIN"
echo ""

mv terraform-provider-datadog_${XC_OS}_${XC_ARCH} $DEST_PATH/$DEST_BIN

echo "Resulting binary: "
echo ""
echo $(ls -la $DEST_PATH/$DEST_BIN)
