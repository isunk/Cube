FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN sed -i '/go RunMonitor()/d' ./main.go
RUN make build CDN=0

FROM frolvlad/alpine-glibc:latest
WORKDIR /app
RUN mkdir -p /data && ln -s /data/cube.db ./cube.db && ln -s /data ./files
COPY --from=builder /app/cube .
COPY ./docs ./docs
ENTRYPOINT ["./cube"]
CMD ["-p", "8090", "-n", "256"]
