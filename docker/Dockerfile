FROM registry.cn-hangzhou.aliyuncs.com/rdc-incubator/builder
ENV GO111MODULE on

RUN mkdir "/usr/sbin/isc-route-service"
COPY ../bin /usr/sbin/isc-route-service

RUN chmod 755 /usr/sbin/isc-route-service/server

CMD ["/usr/sbin/isc-route-service/server"]