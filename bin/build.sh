TAG=$(echo "$TRAVIS_BRANCH" | sed "s/\//-/")
TAG=${TAG:-latest}
echo "$TAG"
export GO111MODULE ON

GOARCH=amd64 GOOS=linux go build -gcflags "all=-N -l" -ldflags "-s -w" -o docker/bin/server main.go
