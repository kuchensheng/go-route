TAG=${1:-latest}
echo "TAG=$TAG"

docker build -t 10.30.30.22:9080/isyscore/isc-route-service:$TAG . \
&& docker push 10.30.30.22:9080/isyscore/isc-route-service:$TAG
