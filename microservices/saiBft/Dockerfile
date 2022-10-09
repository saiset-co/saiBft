FROM ubuntu

WORKDIR /srv

COPY ./build /srv/
RUN apt-get update && apt-get install wget -y

RUN chmod +x ./build/sai-bft
CMD ./sai-bft

EXPOSE 8017
EXPOSE 8018
EXPOSE 8019
