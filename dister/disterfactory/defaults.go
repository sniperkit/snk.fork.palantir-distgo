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

package disterfactory

import (
	"github.com/sniperkit/snk.fork.palantir-distgo/dister/osarchbin"
	"github.com/sniperkit/snk.fork.palantir-distgo/distgo/config"
)

func DefaultConfig() (config.DisterConfig, error) {
	defaultType := osarchbin.TypeName
	return config.DisterConfig{
		Type: &defaultType,
	}, nil
}
