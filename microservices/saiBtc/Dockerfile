FROM ubuntu

WORKDIR /srv

COPY ./build /srv/
RUN apt-get update && apt-get install wget -y

RUN chmod +x ./sai-btc
CMD ./sai-btc

EXPOSE 8017
EXPOSE 8018
EXPOSE 8019
