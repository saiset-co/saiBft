FROM ubuntu

WORKDIR /srv

COPY ./build /srv/
RUN apt-get update && apt-get install wget -y

RUN chmod +x btcvalidator
CMD ./btcvalidator

EXPOSE 3305

