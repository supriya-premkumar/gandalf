FROM spremkumar/gopher-builder:v1.0 as builder

WORKDIR $GOPATH/src/github.com/supriya-premkumar/gandalf
ARG GIT_COMMIT

COPY . .

RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags "-X main.GitCommit=$GIT_COMMIT" \
	-o /bin/gandalf github.com/supriya-premkumar/gandalf

# Compress the binary. Minimal prod image size better for low bandwidth home networks when being pulled
RUN upx /bin/gandalf

FROM alpine:3.11
# Debug utilities. TODO Remove this and build separate prod and debug images
RUN apk update && apk add curl bash
COPY --from=builder /bin/gandalf /gandalf
COPY --from=builder /go/src/github.com/supriya-premkumar/gandalf/deploy/gandalf-config.json /gandalf-config.json
USER nobody

CMD ["/gandalf"]
