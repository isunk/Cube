FROM golang:1.23
WORKDIR /app
COPY . .
# RUN go env -w GO111MODULE=on
# RUN go env -w GOPROXY="https://goproxy.cn,direct"
RUN sed -i '/go RunMonitor()/d' ./main.go
RUN make build CDN=0

FROM alpine
WORKDIR /home
COPY --from=0 cube .
COPY --from=0 docs .
CMD ["./cube"]
