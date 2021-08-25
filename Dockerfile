# 1st stage, build app
FROM golang:latest as builder
RUN apt-get update && apt-get -y upgrade
COPY . /build/app
WORKDIR /build/app

RUN go get ./... && go build -ldflags "-s -w" -o tenderduty main.go

# 2nd stage, create a user to copy, and install libraries needed if connecting to upstream TLS server
FROM debian:10 AS ssl
ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update && apt-get -y upgrade && apt-get install -y ca-certificates && \
    addgroup --gid 26657 --system tenderduty && adduser -uid 26657 --ingroup tenderduty --system --home /var/lib/tenderduty tenderduty

# 3rd and final stage, copy the minimum parts into a scratch container, is a smaller and more secure build. This pulls
# in SSL libraries and CAs so Go can connect to TLS servers.
FROM scratch
COPY --from=ssl /etc/ca-certificates /etc/ca-certificates
COPY --from=ssl /etc/ssl /etc/ssl
COPY --from=ssl /usr/share/ca-certificates /usr/share/ca-certificates
COPY --from=ssl /usr/lib /usr/lib
COPY --from=ssl /lib /lib
COPY --from=ssl /lib64 /lib64

COPY --from=ssl /etc/passwd /etc/passwd
COPY --from=ssl /etc/group /etc/group
COPY --from=ssl --chown=tenderduty:tenderduty /var/lib/tenderduty /var/lib/tenderduty

COPY --from=builder /build/app/tenderduty /bin/tenderduty

USER tenderduty
WORKDIR /var/lib/tenderduty

ENTRYPOINT ["/bin/tenderduty"]