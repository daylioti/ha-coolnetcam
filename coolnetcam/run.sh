#!/usr/bin/with-contenv bashio
export COOLNETCAM_USER="$(bashio::config 'username')"
export COOLNETCAM_PASS="$(bashio::config 'password')"
export COOLNETCAM_LISTEN=":8090"
bashio::log.info "Starting Coolnet Cam (user=${COOLNETCAM_USER}) on :8090"
exec /usr/bin/coolnetcam
