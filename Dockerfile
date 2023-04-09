FROM ubuntu:22.04

RUN apt-get update

# install stuff people might need for their scripts
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y jc jq gron ruby curl mosquitto-clients awscli python3-requests ruby-dev build-essential

RUN curl -sSL https://github.com/mikefarah/yq/releases/download/v4.25.1/yq_linux_amd64 -o /usr/bin/yq
RUN chmod +x /usr/bin/yq

COPY bin/nagentd /usr/bin/nagentd
# RUN curl -sSL https://github.com/penguinpowernz/nagent/releases/download/v1.0.0/nagentd_1.0.0_amd64 -o /usr/bin/nagentd
RUN chmod +x /usr/bin/nagentd

# install stuff people might need for their ruby scripts
RUN gem install influxdb-client influxdb-ruby nats-pure httparty rest-client -N

COPY lib /usr/share/nagentd
COPY scripts/entrypoint.sh /entrypoint.sh

WORKDIR /var/lib/nagentd
VOLUME /var/lib/nagentd

EXPOSE 8080

CMD /entrypoint.sh