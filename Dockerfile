FROM golang:1.20.2 as builder

WORKDIR /app

# copy modules manifests
COPY . .

RUN ls

# build
RUN go build -o lease-based-le

FROM gcr.io/distroless/base:nonroot AS deployable

USER nobody

EXPOSE 8881

COPY --from=builder --chown=nobody:nobody /app/lease-based-le .

ENTRYPOINT ["./lease-based-le"]
