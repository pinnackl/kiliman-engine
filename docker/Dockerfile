FROM node:latest
MAINTAINER titou  "titou@skale-5.com"


RUN git clone https://github.com/pinnackl/kiliman-horizon.git /srv/horizon
WORKDIR /srv/horizon
RUN npm install && npm install -g chateau && npm install -g horizon
ADD . /srv/horizon

CMD []
ENTRYPOINT [ "node", "index.js", "&&", "chateau"]