FROM scratch
ARG GOARCH=amd64

ADD bin/linux/${GOARCH}/bnManager /app/

ENTRYPOINT ["/app/bnManager"]
