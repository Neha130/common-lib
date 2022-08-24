/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package pubsub_lib

import (
	"encoding/json"
	"time"

	"github.com/caarlos0/env"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type NatsClient struct {
	logger                     *zap.SugaredLogger
	JetStrCtxt                 nats.JetStreamContext
	streamConfig               *nats.StreamConfig
	NatsMsgProcessingBatchSize int
	Conn                       nats.Conn
}

type NatsClientConfig struct {
	NatsServerHost             string `env:"NATS_SERVER_HOST" envDefault:"nats://devtron-nats.devtroncd:4222"`
	NatsMsgProcessingBatchSize int    `env:"NATS_MSG_PROCESSING_BATCH_SIZE" envDefault:"2"`
	NatsStreamConfig           string `env:"NATS_STREAM_CONFIG" envDefault:"{}"`
}

/* #nosec */
func NewNatsClient(logger *zap.SugaredLogger) (*NatsClient, error) {

	cfg := &NatsClientConfig{}
	err := env.Parse(cfg)
	if err != nil {
		logger.Error("err", err)
		return &NatsClient{}, err
	}

	configJson := cfg.NatsStreamConfig
	streamCfg := &nats.StreamConfig{}
	if configJson != "" {
		err := json.Unmarshal([]byte(configJson), streamCfg)
		if err != nil {
			logger.Errorw("error occurred while parsing streamConfigJson ", "configJson", configJson, "reason", err)
		}
	}
	logger.Infow("nats config loaded", "config", streamCfg)

	//Connect to NATS
	nc, err := nats.Connect(cfg.NatsServerHost,
		nats.ReconnectWait(10*time.Second), nats.MaxReconnects(100),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Errorw("Nats Connection got disconnected!", "Reason", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Infow("Nats Connection got reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Errorw("Nats Client Connection closed!", "Reason", nc.LastError())
		}))
	if err != nil {
		logger.Error("err", err)
		return &NatsClient{}, err
	}

	//Create a jetstream context
	js, err := nc.JetStream()

	if err != nil {
		logger.Errorw("Error while creating jetstream context", "error", err)
	}

	natsClient := &NatsClient{
		logger:                     logger,
		JetStrCtxt:                 js,
		streamConfig:               streamCfg,
		NatsMsgProcessingBatchSize: cfg.NatsMsgProcessingBatchSize,
	}
	return natsClient, nil
}
