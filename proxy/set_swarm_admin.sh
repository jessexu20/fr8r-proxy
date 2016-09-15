#!/bin/bash

function change_terminal_title {
    local title="$1"
    case $TERM in
        xterm*|rxvt*)
            echo -n -e "\033]0;$title\007"
            ;;
        *)
            ;;
    esac
}

# use 4375 port for accessing the swarm directly
# use 2375 port for accessing the VIP, HAproxy
export DOCKER_HOST=192.168.10.2:2375
#export space=f7f413cb-a678-412d-b024-8e17e28bcb88
export DOCKER_TLS_VERIFY=0
export DOCKER_CONFIG=config/admin-swarm/

change_terminal_title "Admin swarm local"

echo "Use:  DOCKER_TLS_VERIFY=\"\" docker ps"
