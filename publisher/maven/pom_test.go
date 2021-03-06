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

package maven

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sniperkit/snk.fork.palantir-distgo/distgo"
)

func TestRenderPOM(t *testing.T) {
	for i, tc := range []struct {
		name          string
		productID     distgo.ProductID
		version       string
		groupID       string
		packagingType string
		want          string
	}{
		{
			"render POM without packaging",
			"foo",
			"1.0.0",
			"com.palantir",
			"",
			`<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>

  <groupId>com.palantir</groupId>
  <artifactId>foo</artifactId>
  <version>1.0.0</version>
</project>
`,
		},
		{
			"render POM with packaging",
			"foo",
			"1.0.0",
			"com.palantir",
			"tgz",
			`<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <modelVersion>4.0.0</modelVersion>

  <groupId>com.palantir</groupId>
  <artifactId>foo</artifactId>
  <version>1.0.0</version>
  <packaging>tgz</packaging>
</project>
`,
		},
	} {
		got, err := renderPOM(tc.productID, tc.version, tc.groupID, tc.packagingType)
		require.NoError(t, err, "Case %d", i)
		assert.Equal(t, tc.want, got, "Case %d: %s\nOutput:\n%s", i, tc.name, got)
	}
}
