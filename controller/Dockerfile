FROM alpine:latest

RUN apk add --update git go musl-dev

RUN adduser -D vault
USER vault

RUN mkdir -p /home/vault/controller
WORKDIR /home/vault/controller

COPY --chown=vault:vault go.mod .
COPY --chown=vault:vault go.sum .
RUN go mod download

COPY --chown=vault:vault . .
RUN go build

CMD /home/vault/controller/controller
