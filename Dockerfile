FROM alpine:latest

RUN apk update
RUN apk add --no-cache tzdata wget make gcc musl-dev linux-headers ca-certificates go

# glibc
RUN wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub
RUN wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.28-r0/glibc-2.28-r0.apk
RUN apk add glibc-2.28-r0.apk
RUN rm -rf glibc-2.28-r0.apk

# go
RUN mkdir -p /go/src /go/bin && chmod -R 777 /go
ENV GOPATH /go
ENV PATH /go/bin:$PATH
RUN mkdir -p ${GOPATH}/src ${GOPATH}/bin
WORKDIR /go

ENV DATA_HOME /data

ADD . /bcos
RUN cd /bcos && make all
RUN mkdir -p $DATA_HOME/bcos


RUN cp -rf /bcos/build/bin/* /usr/local/bin/
RUN chmod +x /usr/local/bin/*

RUN mkdir -p /example
RUN cp -rf /bcos/docker/data/* /example/
RUN ls /example/
RUN cp -rf /bcos/docker/sh/* /bin/
RUN chmod +x /bin/*

RUN sh /bin/init.sh


RUN rm -rf /bcos

VOLUME /data
EXPOSE 9545 3000 30303 30303/udp

ENTRYPOINT ["/bin/entrypoint.sh"]
CMD ["start", "-D"]
