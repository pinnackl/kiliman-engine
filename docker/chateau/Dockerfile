FROM node:latest

MAINTAINER titou "titou@skale-5.com"

RUN git clone git@github.com:neumino/chateau /srv/chateau
WORKDIR /srv/chateau
RUN npm install
ADD . /srv/horizon

EXPOSE 3000

ENTRYPOINT ["chateau"]