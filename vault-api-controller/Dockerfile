FROM alpine:latest

RUN apk add --update git go

RUN adduser -D vault
USER vault

RUN mkdir -p /home/vault/vault-api-controller
WORKDIR /home/vault/vault-api-controller

COPY --chown=vault:vault go.mod .
COPY --chown=vault:vault go.sum .
RUN go mod download

COPY --chown=vault:vault main.go .
COPY --chown=vault:vault config .
RUN go build

CMD /home/vault/vault-api-controller/vault-api-controller
