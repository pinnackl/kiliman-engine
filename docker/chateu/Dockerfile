FROM node:latest

MAINTAINER titou "titou@skale-5.com"

RUN git clone https://github.com/pinnackl/kiliman-horizon.git /srv/horizon
WORKDIR /srv/horizon
RUN npm install && npm install -g chateau
ADD . /srv/horizon

EXPOSE 3000

ENTRYPOINT ["chateau"]