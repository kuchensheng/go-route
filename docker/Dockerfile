FROM ubuntu:20.04
ENV GO111MODULE on

COPY bin/ /home/isc-route-service

WORKDIR "/home/isc-route-service"

RUN chmod 755 "/home/isc-route-service/"

CMD ["/home/isc-route-service/server"]