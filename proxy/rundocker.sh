#!/bin/bash
set -v
cd dockerize
#HOSTS=$(cat hosts)
#echo $HOSTS
CMD="docker run -v `pwd`:/opt/tls_certs -p 8087:8087 --name hjproxy proxy"
echo $CMD
$(echo $CMD)