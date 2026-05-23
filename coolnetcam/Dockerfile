ARG BUILD_FROM

# ---- build stage: compile coolnetcam natively for the target arch ----
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod ./
COPY main.go index.html setup.html ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /coolnetcam .

# ---- runtime stage: Home Assistant add-on base (s6 + bashio) ----
FROM ${BUILD_FROM}
COPY --from=builder /coolnetcam /usr/bin/coolnetcam
COPY run.sh /run.sh
RUN chmod a+x /run.sh
CMD [ "/run.sh" ]
