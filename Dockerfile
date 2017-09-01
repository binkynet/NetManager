FROM scratch

ADD ./bnManager /app/

ENTRYPOINT ["/app/bnManager"]
