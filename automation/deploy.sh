#!/bin/bash

export ANSIBLE_HOST_KEY_CHECKING=False
export VAULT_PASSWORD_FILE="~/.pwd"

if [ $# -lt 2 ]; then
    echo "Usage: $0 inventory playbook-1.yml [playbook-2.yml ... ]"
    echo "Example: $0 inventory_vagrant db.yml loader.yml api.yml"
    exit 1
fi

cd `dirname $0` && 
ansible-playbook -v --vault-password-file=$VAULT_PASSWORD_FILE --ssh-extra-args="-o ControlMaster=no -o ControlPath=none -o ControlPersist=no" -e "inventory=$1" -i $1 ${@:2}
