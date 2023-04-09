#!/bin/bash

[[ -d /var/lib/nagentd/parsers ]] || cp -Rvf /usr/share/nagentd/parsers /var/lib/nagentd
[[ -d /var/lib/nagentd/sinks ]]   || mkdir -p /var/lib/nagentd/sinks # cp -Rvf /usr/share/nagentd/lib/sinks /var/lib/nagentd
[[ -d /var/lib/nagentd/pre ]]     || mkdir -p /var/lib/nagentd/pre # cp -Rvf /usr/share/nagentd/lib/pre /var/lib/nagentd
[[ -d /var/lib/nagentd/shadows ]] || mkdir -p /var/lib/nagentd/shadows

/usr/bin/nagentd