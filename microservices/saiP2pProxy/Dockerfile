FROM ubuntu

WORKDIR /srv

COPY ./build /srv/
RUN apt-get update && apt-get install wget -y

RUN chmod +x ./sai-p2p
CMD ./sai-p2p

EXPOSE 8071