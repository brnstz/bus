#!/bin/bash

export ANSIBLE_HOST_KEY_CHECKING=False

if [ $# -lt 2 ]; then
    echo "Usage: $0 inventory playbook-1.yml [playbook-2.yml ... ]"
    echo "Example: $0 inventory_vagrant api.yml"
    exit 1
fi

ansible-playbook -v -i $1 ${@:2}
