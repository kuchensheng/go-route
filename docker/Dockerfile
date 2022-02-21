FROM ubuntu:20.04
ENV GO111MODULE on

RUN mkdir "/usr/sbin/isc-route-service"

COPY bin/ /usr/sbin/isc-route-service/bin

WORKDIR "/usr/sbin/isc-route-service/bin"

RUN chmod 755 /usr/sbin/isc-route-service/bin/server

CMD ["/usr/sbin/isc-route-service/bin/server"]