FROM ubuntu

WORKDIR /srv

COPY ./build /srv/
RUN apt-get update && apt-get install wget curl -y

RUN chmod +x p2p_lin64
CMD ./p2p_lin64

EXPOSE 8112
