FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
# RUN go env -w GO111MODULE=on
# RUN go env -w GOPROXY="https://goproxy.cn,direct"
RUN sed -i '/go RunMonitor()/d' ./main.go
RUN make build CDN=0

FROM frolvlad/alpine-glibc:latest
WORKDIR /app
COPY --from=builder /app/cube .
COPY ./docs ./docs
ENTRYPOINT ["./cube"]
CMD ["-p", "8090", "-n", "256"]
