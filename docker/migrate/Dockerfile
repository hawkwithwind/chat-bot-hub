FROM migrate/migrate

ARG mirror=mirrors.ustc.edu.cn
ADD modifysource.sh /home/scripts/modifysource.sh
RUN /bin/sh /home/scripts/modifysource.sh $mirror
RUN cat /etc/apk/repositories

RUN apk add --no-cache --update 'su-exec>=0.2'
RUN apk add --no-cache --update shadow

COPY docker-entrypoint.sh /usr/local/bin/

RUN groupadd -r migrateuser && useradd --no-log-init -r -g migrateuser migrateuser

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["--help"]