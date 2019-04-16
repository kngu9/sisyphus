// Copyright 2019 CanonicalLtd

package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/juju/errgo"
	"go.uber.org/zap/zapcore"
)

// LogLevel returns the level of logging to perform. If the
// environment variable is not set, the level will be the default
// INFO level.
func LogLevel() zapcore.Level {
	var level zapcore.Level
	level.UnmarshalText([]byte(os.Getenv("LOGLEVEL")))
	return level
}

// Config returns the CONFIG environment variable.
func Config() string {
	return os.Getenv("CONFIG")
}

func KafkaClientID() string {
	id := os.Getenv("KAFKA_CLIENT_ID")
	if id == "" {
		return "sisyphus_simulation"
	}
	return id
}

func KafkaVersion() (sarama.KafkaVersion, error) {
	version := os.Getenv("KAFKA_VERSION")
	if version == "" {
		return sarama.V2_0_0_0, nil
	}
	return sarama.ParseKafkaVersion(version)
}

// KafkaBrokerURLs returns the KAFKA_BROKERS environment variable
func KafkaBrokerURLs() []string {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		return nil
	}
	return strings.Split(brokers, ",")
}

// TLSConfig contains values a client needs to connect to a server via tls.
type TLSConfig struct {
	// Certificate holds the client certificate.
	Certificate *tls.Certificate
	// CACertificate holds the CA certificate of the authority
	// that created the client certificate.
	CACertificate *x509.Certificate
}

// Config loads specified certificates and returns tls.Config.
func (c *TLSConfig) Config() (*tls.Config, error) {
	if c.Certificate != nil {
		config := &tls.Config{
			Certificates: []tls.Certificate{*c.Certificate},
		}
		if c.CACertificate != nil {
			caCertPool := x509.NewCertPool()
			caCertPool.AddCert(c.CACertificate)
			config.RootCAs = caCertPool
		}
		return config, nil
	}
	return nil, errgo.New("certificate not specified")
}

// KafkaTLS fetches KAFKA_CLIENT_CERT, KAFKA_CLIENT_KEY and KAFKA_CA_CERT
// environment variables and returns a tls config structure.
func KafkaTLS() (*TLSConfig, error) {
	clientCertString := os.Getenv("KAFKA_CLIENT_CERT")
	clientKeyString := os.Getenv("KAFKA_CLIENT_KEY")
	caCertString := os.Getenv("KAFKA_CA_CERT")

	if clientCertString == "" && clientKeyString == "" {
		return nil, nil
	}
	cert, err := tls.X509KeyPair([]byte(clientCertString), []byte(clientKeyString))
	if err != nil {
		return nil, errgo.Mask(err)
	}
	config := TLSConfig{
		Certificate: &cert,
	}
	if caCertString != "" {
		pemData, _ := pem.Decode([]byte(caCertString))
		if pemData == nil {
			return nil, errgo.New("failed to decode CA certificate")
		}
		caCert, err := x509.ParseCertificate(pemData.Bytes)
		if err != nil {
			return nil, errgo.NoteMask(err, "invalid CA certificate")
		}
		config.CACertificate = caCert
	}
	return &config, nil
}
