ARG ARCH
FROM golang:1.17
WORKDIR /build
ADD . .
ARG ARCH
RUN make build.$ARCH

FROM $ARCH/alpine:3.15
COPY --from=0 /build/ping-exporter /bin/ping-exporter

CMD ["ping-expoter"]
EXPOSE 8080
