// Copyright 2019 CanonicalLtd

package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/juju/errors"
	"go.uber.org/zap/zapcore"
)

var (
	logLevel            string
	configPath          string
	kafkaClientID       string
	kafkaBrokers        string
	kafkaClientCertPath string
	kafkaClientKeyPath  string
	kafkaCAPath         string
	workers             int
)

func init() {
	flag.StringVar(&logLevel, "log-level", "", "logging level")
	flag.StringVar(&configPath, "config", "", "configuration file path")
	flag.StringVar(&kafkaClientID, "kafka-client-id", "", "kafka client id")
	flag.StringVar(&kafkaBrokers, "kafka-brokers", "", "kafka brokers")
	flag.StringVar(&kafkaClientCertPath, "kafka-cert", "", "kafka client cert")
	flag.StringVar(&kafkaClientKeyPath, "kafka-key", "", "kafka client key")
	flag.StringVar(&kafkaCAPath, "kafka-ca", "", "kafka ca cert")
	flag.IntVar(&workers, "workers", 0, "number of workers")
}

func Workers() (int, error) {
	if workers != 0 {
		return workers, nil
	}
	if number := os.Getenv("WORKERS"); number != "" {
		n, err := strconv.Atoi(number)
		if err != nil {
			return 0, errors.Trace(err)
		}
		return n, nil
	}
	return 16, nil
}

// LogLevel returns the level of logging to perform. If the
// environment variable is not set, the level will be the default
// INFO level.
func LogLevel() zapcore.Level {
	var level zapcore.Level
	if logLevel != "" {
		level.UnmarshalText([]byte(logLevel))

	} else {
		level.UnmarshalText([]byte(os.Getenv("LOGLEVEL")))
	}
	return level
}

// Config returns the CONFIG environment variable.
func Config() string {
	if configPath != "" {
		return configPath
	}
	return os.Getenv("CONFIG")
}

func KafkaClientID() string {
	if kafkaClientID != "" {
		return kafkaClientID
	}
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
	if kafkaBrokers != "" {
		return strings.Split(kafkaBrokers, ",")
	}
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
	return nil, errors.New("certificate not specified")
}

func kafkaCert() (string, error) {
	if kafkaClientCertPath != "" {
		cert, err := ioutil.ReadFile(kafkaClientCertPath)
		if err != nil {
			return "", errors.Trace(err)
		}
		return string(cert), nil
	}
	return os.Getenv("KAFKA_CLIENT_CERT"), nil
}

func kafkaKey() (string, error) {
	if kafkaClientKeyPath != "" {
		key, err := ioutil.ReadFile(kafkaClientKeyPath)
		if err != nil {
			return "", errors.Trace(err)
		}
		return string(key), nil
	}
	return os.Getenv("KAFKA_CLIENT_KEY"), nil
}

func kafkaCA() (string, error) {
	if kafkaCAPath != "" {
		cert, err := ioutil.ReadFile(kafkaCAPath)
		if err != nil {
			return "", errors.Trace(err)
		}
		return string(cert), nil
	}
	return os.Getenv("KAFKA_CA_CERT"), nil
}

// KafkaTLS fetches KAFKA_CLIENT_CERT, KAFKA_CLIENT_KEY and KAFKA_CA_CERT
// environment variables and returns a tls config structure.
func KafkaTLS() (*TLSConfig, error) {
	clientCertString, err := kafkaCert()
	if err != nil {
		return nil, errors.Trace(err)
	}
	clientKeyString, err := kafkaKey()
	if err != nil {
		return nil, errors.Trace(err)
	}
	caCertString, err := kafkaCA()
	if err != nil {
		return nil, errors.Trace(err)
	}

	if clientCertString == "" && clientKeyString == "" {
		return nil, nil
	}

	cert, err := tls.X509KeyPair([]byte(clientCertString), []byte(clientKeyString))
	if err != nil {
		return nil, errors.Annotate(err, "failed to parse the keypair")
	}
	config := TLSConfig{
		Certificate: &cert,
	}
	if caCertString != "" {
		pemData, _ := pem.Decode([]byte(caCertString))
		if pemData == nil {
			return nil, errors.New("failed to decode CA certificate")
		}
		caCert, err := x509.ParseCertificate(pemData.Bytes)
		if err != nil {
			return nil, errors.Annotate(err, "invalid CA certificate")
		}
		config.CACertificate = caCert
	}
	return &config, nil
}
