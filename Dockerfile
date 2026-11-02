FROM scratch

COPY manta-cli /manta-cli

USER 65532:65532

ENTRYPOINT ["/manta-cli"]
