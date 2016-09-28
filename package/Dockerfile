FROM busybox

ADD ./secrets-bridge /secrets-bridge
ADD ./ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/secrets-bridge"]
CMD ["-h"]
