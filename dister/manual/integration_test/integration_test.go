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

package integration_test

import (
	"fmt"
	"path"
	"testing"

	"github.com/nmiyake/pkg/gofiles"
	"github.com/palantir/godel/framework/pluginapitester"
	"github.com/palantir/godel/pkg/products/v2/products"
	"github.com/palantir/pkg/specdir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sniperkit/snk.fork.palantir-distgo/dister/distertester"
)

func TestManualDist(t *testing.T) {
	const godelYML = `exclude:
  names:
    - "\\..+"
    - "vendor"
  paths:
    - "godel"
`

	pluginPath, err := products.Bin("dist-plugin")
	require.NoError(t, err)

	distertester.RunAssetDistTest(t,
		pluginapitester.NewPluginProvider(pluginPath),
		nil,
		[]distertester.TestCase{
			{
				Name: "manual creates output with script",
				Specs: []gofiles.GoFileSpec{
					{
						RelPath: "foo/foo.go",
						Src:     `package main; func main() {}`,
					},
				},
				ConfigFiles: map[string]string{
					"godel/config/godel.yml": godelYML,
					"godel/config/dist-plugin.yml": `
products:
  foo:
    build:
      main-pkg: ./foo
      os-archs:
        - os: darwin
          arch: amd64
        - os: linux
          arch: amd64
    dist:
      disters:
        type: manual
        config:
          extension: tar
        script: |
                #!/usr/bin/env bash
                echo "hello" > $DIST_WORK_DIR/out.txt
                tar -cf "$DIST_DIR/$DIST_NAME".tar -C "$DIST_WORK_DIR" out.txt
`,
				},
				WantOutput: func(projectDir string) string {
					return `Creating distribution for foo at out/dist/foo/1.0.0/manual/foo-1.0.0.tar
Finished creating manual distribution for foo
`
				},
				Validate: func(projectDir string) {
					wantLayout := specdir.NewLayoutSpec(
						specdir.Dir(specdir.LiteralName("1.0.0"), "",
							specdir.Dir(specdir.LiteralName("manual"), "",
								specdir.Dir(specdir.LiteralName("foo-1.0.0"), "",
									specdir.File(specdir.LiteralName("out.txt"), ""),
								),
								specdir.File(specdir.LiteralName("foo-1.0.0.tar"), ""),
							),
						), true,
					)
					assert.NoError(t, wantLayout.Validate(path.Join(projectDir, "out", "dist", "foo", "1.0.0"), nil))
				},
			},
			{
				Name: "manual fails if script does not create expected output",
				Specs: []gofiles.GoFileSpec{
					{
						RelPath: "foo/foo.go",
						Src:     `package main; func main() {}`,
					},
				},
				ConfigFiles: map[string]string{
					"godel/config/godel.yml": godelYML,
					"godel/config/dist-plugin.yml": `
products:
  foo:
    build:
      main-pkg: ./foo
      os-archs:
        - os: darwin
          arch: amd64
        - os: linux
          arch: amd64
    dist:
      disters:
        type: manual
        config:
          extension: zip
`,
				},
				WantOutput: func(projectDir string) string {
					return fmt.Sprintf(`Creating distribution for foo at out/dist/foo/1.0.0/manual/foo-1.0.0.zip
Error: dist failed for foo: expected output does not exist at %s/out/dist/foo/1.0.0/manual/foo-1.0.0.zip: stat %s/out/dist/foo/1.0.0/manual/foo-1.0.0.zip: no such file or directory
`, projectDir, projectDir)
				},
				WantError: true,
			},
		},
	)
}

func TestManualUpgradeConfig(t *testing.T) {
	pluginPath, err := products.Bin("dist-plugin")
	require.NoError(t, err)

	pluginapitester.RunUpgradeConfigTest(t,
		pluginapitester.NewPluginProvider(pluginPath),
		nil,
		[]pluginapitester.UpgradeConfigTestCase{
			{
				Name: `legacy configuration is upgraded`,
				ConfigFiles: map[string]string{
					"godel/config/dist.yml": `
products:
  foo:
    build:
      main-pkg: ./foo
      os-archs:
        - os: darwin
          arch: amd64
        - os: linux
          arch: amd64
    dist:
      dist-type:
        type: manual
        info:
          extension: zip
`,
				},
				Legacy: true,
				WantOutput: `Upgraded configuration for dist-plugin.yml
`,
				WantFiles: map[string]string{
					"godel/config/dist-plugin.yml": `products:
  foo:
    build:
      main-pkg: ./foo
      os-archs:
      - os: darwin
        arch: amd64
      - os: linux
        arch: amd64
    dist:
      disters:
        manual:
          type: manual
          config:
            extension: zip
`,
				},
			},
			{
				Name: `valid v0 config works`,
				ConfigFiles: map[string]string{
					"godel/config/dist-plugin.yml": `
products:
  foo:
    build:
      main-pkg: ./foo
      os-archs:
        - os: darwin
          arch: amd64
        - os: linux
          arch: amd64
    dist:
      disters:
        type: manual
        config:
          # comment
          extension: zip
`,
				},
				WantOutput: ``,
				WantFiles: map[string]string{
					"godel/config/dist-plugin.yml": `
products:
  foo:
    build:
      main-pkg: ./foo
      os-archs:
        - os: darwin
          arch: amd64
        - os: linux
          arch: amd64
    dist:
      disters:
        type: manual
        config:
          # comment
          extension: zip
`,
				},
			},
		},
	)
}
