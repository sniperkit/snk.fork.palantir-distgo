/*
Sniperkit-Bot
- Status: analyzed
*/

// Copyright 2016 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v0

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/sniperkit/snk.fork.palantir-distgo/publisher"
)

type Config struct {
	publisher.BasicConnectionInfo `yaml:",inline,omitempty"`
	Repository                    string `yaml:"repository,omitempty"`
	NoPOM                         bool   `yaml:"no-pom,omitempty"`
}

func UpgradeConfig(cfgBytes []byte) ([]byte, error) {
	var cfg Config
	if err := yaml.UnmarshalStrict(cfgBytes, &cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal artifactory publisher v0 configuration")
	}
	return cfgBytes, nil
}
