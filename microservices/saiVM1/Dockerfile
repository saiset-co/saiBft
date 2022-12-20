FROM ubuntu

WORKDIR /srv

COPY ./build /srv/
RUN apt-get update && apt-get install wget -y

RUN chmod +x ./sai-vm1
CMD ./sai-vm1 start

EXPOSE 8009
EXPOSE 8089
EXPOSE 8099
