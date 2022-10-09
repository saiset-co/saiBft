FROM ubuntu

WORKDIR /srv

COPY ./build /srv/
RUN apt-get update && apt-get install -y wget locales curl && locale-gen ${OS_LOCALE}
RUN wget http://archive.ubuntu.com/ubuntu/pool/main/o/openssl/libssl1.1_1.1.0g-2ubuntu4_amd64.deb
RUN dpkg -i libssl1.1_1.1.0g-2ubuntu4_amd64.deb
RUN dpkg-reconfigure locales && apt-get install -y sudo gnupg \
 && apt-get install -y --reinstall ca-certificates \
 && curl -fsSL https://www.mongodb.org/static/pgp/server-4.4.asc | apt-key add \
 && echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/4.4 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-4.4.list \
 && apt-get update && apt-get install -y mongodb-org \
 && apt-get purge -y --auto-remove gnupg \
 && rm -rf /var/lib/apt/lists/*

RUN openssl req -x509 -nodes -newkey rsa:2048 -keyout server.rsa.key -out server.rsa.crt -days 3650 -subj '/CN=localhost,sai-storage'
RUN ln -sf server.rsa.key server.key
RUN ln -sf server.rsa.crt server.crt

VOLUME ["/data/db"]

RUN chmod +x sai-storage
CMD ./sai-storage

EXPOSE 27017
EXPOSE 8801
EXPOSE 8802
