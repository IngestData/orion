FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/orion ./src

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/orion /orion
EXPOSE 8090
ENTRYPOINT ["/orion"]
