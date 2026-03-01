# Distroless dockerfile for auth server
FROM golang:1.24 AS build

WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN CGO_ENABLED=1 go build -o /go/bin/durin .

FROM gcr.io/distroless/base-debian12
COPY --from=build /go/bin/durin /
CMD ["/durin"]
