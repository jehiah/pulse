FROM alpine:3.2

RUN apk add -U gpgme curl mtr && rm -rf /var/cache/apk/*
RUN mkdir /minion-dir
ADD	./docker-entry.sh /bin/docker-entry.sh
ADD	./deploy-minion.sh /bin/deploy-minion.sh

ENTRYPOINT ["/bin/docker-entry.sh"]