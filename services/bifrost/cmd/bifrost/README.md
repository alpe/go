# Bifrost
 
 
## Build docker artifact

* Build static binary
```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -tags nocgo -ldflags '-extldflags "-static"' \
         github.com/stellar/go/services/bifrost/cmd/bifrost
```
* Build docker
```bash

docker build -t mybifrost -f ./services/bifrost/cmd/bifrost/Dockerfile .
docker tag mybifrost $DOCKER_ID_USER/mybifrost
docker push $DOCKER_ID_USER/mybifrost
```

TODO: upgrade build container
```bash
docker run --rm \
  --env SKIP_TESTS=yes \
  --env CGO_ENABLED=yes \
  --volume $(pwd):/src \
  --volume /var/run/docker.sock:/var/run/docker.sock \
  alpebuild/golang-builder:next_version mybifrost
```