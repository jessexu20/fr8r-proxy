# Fr8r Proxy images and how to build them


## fr8r-proxy

```bash
docker login (with dockerhub creds)
cd fr8r-proxy
# to build for development (large image, fast compile):
./builddocker.sh -f Dockerfile.dev
# to build for deployment (smaller image, longer compile):
./buildocker.sh

# see your new image:
docker images
docker tag api-proxy fr8r/api-proxy
docker push fr8r/api-proxy
```

