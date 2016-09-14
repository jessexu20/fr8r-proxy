# ContainerCafe images and how to build them

## openradiant/remoteabac

``` bash
cd openradiant/remoteabac
docker login (with dockerhub creds)
docker build -t openradiant/remoteabac .
docker push openradiant/remoteabac
```

## containercafe/proxy

```bash
docker login (with dockerhub creds)
cd proxy
./builddocker.sh
# see your new image:
docker images
docker tag api-proxy containercafe/api-proxy
docker push containercafe/api-proxy
```

## openradiant/km

Building is straightforward:

``` bash
cd openradiant/misc/dockerfiles/km
docker login (with dockerhub creds)
docker build -t openradiant/km:$tag .
docker push openradiant/km:$tag
```

... where `$tag` is the value of `K8S_BRANCH` in the Dockerfile ---
i.e., the git tag or branch of
https://github.com/ibm-contribs/kubernetes.git that is being built.

The selection of *what* to build is more complicated.  The choice is
in the Dockerfile.  Documentation needed for the why behind that
choice.
