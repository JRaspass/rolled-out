FROM golang:1.21.1 AS dev

ENV CGO_ENABLED=0 GOAMD64=v4 GOPATH=

RUN apt-get update && apt-get install -y reflex

CMD reflex -sd none -r '\.(go|html|json)$' -- go run .

###

FROM dev AS build

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN go build -ldflags -s -trimpath

###

FROM scratch AS live

COPY --from=build /go/rolled-out                     /
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD ["/rolled-out"]
