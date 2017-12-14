FROM scratch
MAINTAINER Alex Peters <info@alexanderpeters.de>

ENV serviceRoot=/services/bifrost
COPY ${serviceRoot}/database/migrations/ /migrations/
COPY ${serviceRoot}/cmd/bifrost/bifrost /
COPY ${serviceRoot}/cmd/bifrost/cacert.pem /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/bifrost"]
CMD [""]