#!/bin/bash

# clear_ipc
#
# Delete the IPC objects created by a user. The user and IPC type are specified on the command
# line.
#
# usage: clear_ipc username type

USERNAME=$1

# Type of IPC object. Possible values are:
#   q -- message queue
#   m -- shared memory
#   s -- semaphore
TYPE=$2

ipcs -$TYPE | grep $USERNAME | awk ' { print $2 } ' | xargs -I {} ipcrm -$TYPE {}