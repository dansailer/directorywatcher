FROM golang:1.8 as builder
RUN set -x \
    && curl https://glide.sh/get | sh
WORKDIR /go/src/github.com/dansailer/directorywatcher/
COPY [ "glide.yaml", "glide.lock", "/go/src/github.com/dansailer/directorywatcher/"]
RUN set -x \
    && glide install
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o directorywatcher .

FROM alpine:latest
RUN set -x \
    && apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/dansailer/directorywatcher/directorywatcher .
CMD ["./directorywatcher"]