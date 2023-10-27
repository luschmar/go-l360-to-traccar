FROM alpine as certsrc
RUN apk update && apk upgrade && apk add --no-cache ca-certificates
RUN update-ca-certificates

FROM scratch
COPY --from=certsrc /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ADD go-l360-to-traccar /
CMD ["/go-l360-to-traccar"]