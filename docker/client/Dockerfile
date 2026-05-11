FROM nginx:1.27-alpine

# Только эта переменная участвует в envsubst (остальные $host, $uri и т.д. не трогаем).
ENV NGINX_ENVSUBST_FILTER=^CLIENT_MAX_BODY_SIZE$
ENV CLIENT_MAX_BODY_SIZE=25m

COPY web/static/ /var/www/html/
COPY docker/client/default.conf.template /etc/nginx/templates/default.conf.template

EXPOSE 8000
