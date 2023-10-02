#!/bin/sh
if [ -z "${AWS_LAMBDA_RUNTIME_API}" ]; then
  exec /aws-lambda-rie /main "$@"
else
  exec /main "$@"
fi
