#!/bin/sh
if [ "$FAKEACCEL_SPEC" != "" ]; then
  exec /usr/bin/habanalabs-device-plugin-fake "$@"
else
  exec /usr/bin/habanalabs-device-plugin "$@"
fi
