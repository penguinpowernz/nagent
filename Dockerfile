FROM ubuntu:22.04

RUN apt-get update
RUN apt-get install -y jc jq gron ruby curl mosquitto-clients
RUN curl -sSL https://github.com/mikefarah/yq/releases/download/v4.25.1/yq_linux_amd64 -o /usr/bin/yq
RUN chmod +x /usr/bin/yq

RUN curl -sSL https://github.com/penguinpowernz/nagent/releases/download/v1.0.0/nagentd_1.0.0_amd64 -o /usr/bin/nagentd
RUN chmod +x /usr/bin/nagentd

RUN gem install influxdb-client influxdb-ruby nats-pure -N

COPY lib/* /usr/share/nagentd/lib/
COPY scripts/entrypoint.sh /entrypoint.sh

WORKDIR /var/lib/nagentd
VOLUME /var/lib/nagentd

EXPOSE 8080

CMD /entrypoint.sh