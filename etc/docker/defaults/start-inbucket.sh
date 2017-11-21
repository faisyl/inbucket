#!/bin/sh
# start-inbucket.sh
# description: start inbucket (runs within a docker container)

CONF_SOURCE="$INBUCKET_HOME/defaults"
CONF_TARGET="/con/configuration"
TLS_CERT="/con/configuration/stunnel/server.crt"
TLS_KEY="/con/configuration/stunnel/server.key"

set -eo pipefail

install_default_config() {
  local file="$1"
  local source="$CONF_SOURCE/$file"
  local target="$CONF_TARGET/$file"

  if [ ! -e "$target" ]; then
    echo "Installing default $file to $CONF_TARGET"
    install "$source" "$target"
  fi
}

install_default_config "inbucket.conf"
install_default_config "greeting.html"

if [ -f $TLS_CERT ] && [ -f $TLS_KEY ]; then
  # Runs as daemon in the background
  echo "Starting stunnel to serve HTTPS on port 443"
  chmod 0640 $TLS_KEY
  stunnel
fi

exec "$INBUCKET_HOME/bin/inbucket" $* 
