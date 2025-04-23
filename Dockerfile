FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
# RUN go env -w GO111MODULE=on
# RUN go env -w GOPROXY="https://goproxy.cn,direct"
RUN sed -i '/go RunMonitor()/d' ./main.go
RUN make build CDN=0

FROM alpine-glibc
WORKDIR /home
COPY --from=builder /app/cube .
COPY ./docs ./docs
CMD ["./cube"]
