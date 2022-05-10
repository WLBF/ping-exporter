ARG ARCH
FROM golang:1.17
ENV http_proxy "http://172.17.0.1:7890"
ENV HTTP_PROXY "http://172.17.0.1:7890"
ENV https_proxy "http://172.17.0.1:7890"
ENV HTTPS_PROXY "http://172.17.0.1:7890"
WORKDIR /build
ADD . .
ARG ARCH
RUN make build.$ARCH

FROM $ARCH/alpine:3.15
COPY --from=0 /build/ping-exporter /bin/ping-exporter

CMD ["ping-exporter"]
EXPOSE 8080
