sed -i -e "s/\($1=\).*/\1"\"$2\"/"" Dockerfile
rm -f Dockerfile-e