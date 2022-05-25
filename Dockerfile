FROM ubuntu:18.04

RUN apt-get update
RUN apt-get install -y jc jq gron ruby python curl mosquitto-clients
RUN curl https://github.com/mikefarah/yq/releases/download/v4.25.1/yq_linux_amd64 -o /usr/bin/yq
RUN chmod +x yq

RUN gem install influxdb-client-ruby influxdb-ruby nats-pure -N

COPY lib/* /usr/share/nagentd/lib
COPY bin/nagentd /usr/bin
COPY scripts/entrypoint.sh /usr/bin

WORKDIR /var/lib/nagentd
VOLUME /var/lib/nagentd

EXPOSE 8080

CMD entrypoint.sh