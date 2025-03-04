/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jaeger

import (
	"github.com/mitchellh/mapstructure"

	"github.com/pkg/errors"

	"go.opentelemetry.io/otel/exporters/jaeger"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

import (
	"github.com/apache/dubbo-go-pixiu/pixiu/pkg/model"
)

type jaegerConfig struct {
	Url string `yaml:"url" json:"url" mapstructure:"url"`
}

func NewJaegerExporter(cfg *model.TracerConfig) (sdktrace.SpanExporter, error) {
	var config jaegerConfig
	if err := mapstructure.Decode(cfg.Config, &config); err != nil {
		return nil, errors.Wrap(err, "config error")
	}
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(config.Url)))
	return exp, err
}
