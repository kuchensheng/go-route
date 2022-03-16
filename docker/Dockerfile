FROM ubuntu:20.04
ENV GO111MODULE on
RUN set -x \
    && export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get install -y tzdata \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && rm -rf /var/lib/apt/lists/*

COPY . /home/isc-route-service

WORKDIR "/home/isc-route-service"

RUN chmod 755 "/home/isc-route-service/"

CMD ["/home/isc-route-service/isc-route-service"]