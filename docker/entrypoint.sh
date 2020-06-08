#!/bin/bash

set -e

if [ ! -e choria-provisioner.yaml ]
then
  cp choria-provisioner.yaml.template choria-provisioner.yaml
  echo "Copied provisioner config file template to /etc/choria-provisioner/choria-provisioner.yaml"
fi

if [ ! -e client.cfg ]
then
  cp client.cfg.template client.cfg
  echo "Copied client config file template to /etc/choria-provisioner/client.cfg"
fi

if [ ! -e "/helper/provision-helper" ]
then
  echo "Please provide /helper/provision-helper"
  exit 1
fi

if [ -z "${PROV_TOKEN}" ]
then
  echo "Please set PROV_TOKEN"
  exit 1
fi

if [ -z "${PROVISIONER_PASSWORD}" ]
then
  echo "Please set PROVISIONER_PASSWORD"
  exit 1
fi

if [ -z "${CHORIA_PASSWORD}" ]
then
  echo "Please set CHORIA_PASSWORD"
  exit 1
fi

if [ -z "${CHORIA_BROKER_URL}" ]
then
  echo "Please set CHORIA_BROKER_URL"
  exit 1
fi

if [ "${PROV_INSECURE}" = "false" ]
then
  if [ ! -e "/etc/choria-provisioner/ssl/cert.pem" ]
  then
    echo "Please create provisioner certificate in /etc/choria-provisioner/ssl/cert.pem"
    exit 1
  fi

  if [ ! -e "/etc/choria-provisioner/ssl/key.pem" ]
  then
    echo "Please create provisioner key in /etc/choria-provisioner/ssl/key.pem"
    exit 1
  fi

  if [ ! -e "/etc/choria-provisioner/ssl/ca.pem" ]
  then
    echo "Please create provisioner key in /etc/choria-provisioner/ssl/ca.pem"
    exit 1
  fi
fi

if [ "${PROV_FEATURES_JWT}" = "true" ]
then
  if [ ! -e "/etc/choria-provisioner/ssl/jwt-verify.pem" ]
  then
    echo "Please create /etc/choria-provisioner/ssl/jwt-verify.pem"
    exit 1
  fi
fi

sed -i'' "s/{{WORKERS}}/${PROV_WORKERS}/" choria-provisioner.yaml
sed -i'' "s/{{INTERVAL}}/${PROV_INTERVAL}/" choria-provisioner.yaml
sed -i'' "s/{{LOGLEVEL}}/${PROV_LOGLEVEL}/" choria-provisioner.yaml
sed -i'' "s/{{INSECURE}}/${PROV_INSECURE}/" choria-provisioner.yaml
sed -i'' "s/{{SITE}}/${PROV_SITE}/" choria-provisioner.yaml
sed -i'' "s/{{TOKEN}}/${PROV_TOKEN}/" choria-provisioner.yaml
sed -i'' "s/{{FEATURES_PKI}}/${PROV_FEATURES_PKI}/" choria-provisioner.yaml
sed -i'' "s/{{FEATURES_JWT}}/${PROV_FEATURES_JWT}/" choria-provisioner.yaml
sed -i'' "s/{{FEATURES_BROKER}}/${PROV_FEATURES_BROKER}/" choria-provisioner.yaml
sed -i'' "s/{{PROVISIONER_PASSWORD}}/${PROVISIONER_PASSWORD}/" choria-provisioner.yaml
sed -i'' "s/{{CHORIA_PASSWORD}}/${CHORIA_PASSWORD}/" choria-provisioner.yaml
sed -i'' "s/{{BROKER_URL}}/${CHORIA_BROKER_URL}/" client.cfg
sed -i'' "s/{{PROVISIONER_PASSWORD}}/${PROVISIONER_PASSWORD}/" client.cfg

exec /usr/sbin/choria-provisioner --config choria-provisioner.yaml --choria-config client.cfg
