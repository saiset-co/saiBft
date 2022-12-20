FROM ubuntu

WORKDIR /srv

COPY ./build /srv/
RUN apt-get update && apt-get install wget -y

RUN chmod +x ./sai-bft
#CMD ./sai-bft tx send3 from3 to3 amount3 denom3
CMD ./sai-bft start

EXPOSE 8017
EXPOSE 8018
EXPOSE 8019
