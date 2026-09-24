FROM alpine:3.24.2

RUN addgroup -g 10001 -S app \
 && adduser -S -D -H -h /nonexistent -s /sbin/nologin -G app -u 10001 app

WORKDIR /opt/app
COPY --chown=app:app ./server /opt/app/server
COPY --chown=app:app ./assets /opt/app/assets
RUN chmod 500 /opt/app/server

USER app:app
EXPOSE 8080
ENTRYPOINT ["/opt/app/server"]
