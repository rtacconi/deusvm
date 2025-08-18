FROM golang:1.22 as build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/deusvm ./cmd/deusvm

FROM gcr.io/distroless/base-debian12
COPY --from=build /out/deusvm /deusvm
EXPOSE 8080
ENTRYPOINT ["/deusvm"]


