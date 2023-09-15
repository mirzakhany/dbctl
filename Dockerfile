# Start by building the application.
FROM golang:1.21 as build

WORKDIR /go/src/app
COPY . .

RUN make build

# Copy the binary into a distroless container.
FROM gcr.io/distroless/static-debian11
COPY --from=build /go/src/app/dbctl /
CMD ["/dbctl"]
