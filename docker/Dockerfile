FROM node:latest
MAINTAINER titou  "titou@skale-5.com"


RUN git clone https://github.com/pinnackl/kiliman-horizon.git /srv/horizon
WORKDIR /srv/horizon
RUN npm install
ADD . /srv/horizon

EXPOSE 8181
EXPOSE 28015

CMD []
ENTRYPOINT ["node", "index.js"]