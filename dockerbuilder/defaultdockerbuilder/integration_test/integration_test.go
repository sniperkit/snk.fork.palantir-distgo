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
	"testing"

	"github.com/nmiyake/pkg/gofiles"
	"github.com/palantir/godel/framework/pluginapitester"
	"github.com/palantir/godel/pkg/products/v2/products"
	"github.com/stretchr/testify/require"

	"github.com/sniperkit/snk.fork.palantir-distgo/dockerbuilder/dockerbuildertester"
)

func TestDocker(t *testing.T) {
	const godelYML = `exclude:
  names:
    - "\\..+"
    - "vendor"
  paths:
    - "godel"
`

	pluginPath, err := products.Bin("dist-plugin")
	require.NoError(t, err)

	dockerbuildertester.RunAssetDockerBuilderTest(t,
		pluginapitester.NewPluginProvider(pluginPath),
		nil,
		[]dockerbuildertester.TestCase{
			{
				Name: "builds Docker image",
				Specs: []gofiles.GoFileSpec{
					{
						RelPath: "foo/foo.go",
						Src:     `package main; func main() {}`,
					},
					{
						RelPath: "testContextDir/Dockerfile",
						Src: `FROM alpine:3.5
RUN echo 'Product: \{\{Product\}\}'
RUN echo 'Version: \{\{Version\}\}'
RUN echo 'Repository: \{\{Repository\}\}'
`,
					},
				},
				ConfigFiles: map[string]string{
					"godel/config/godel.yml": godelYML,
					"godel/config/dist-plugin.yml": `
products:
  foo:
    build:
      main-pkg: ./foo
    dist:
      disters:
        type: os-arch-bin
    docker:
      docker-builders:
        tester:
          type: default
          context-dir: testContextDir
          tag-templates:
            - tester-tag:latest-and-greatest
`,
				},
				Args: []string{
					"build",
					"--dry-run",
				},
				WantOutput: func(projectDir string) string {
					return fmt.Sprintf(`[DRY RUN] Running Docker build for configuration tester of product foo...
[DRY RUN] Run [docker build --file %s/testContextDir/Dockerfile -t tester-tag:latest-and-greatest %s/testContextDir]
`, projectDir, projectDir)
				},
			},
			{
				Name: "filters Docker tags based on flag",
				Specs: []gofiles.GoFileSpec{
					{
						RelPath: "foo/foo.go",
						Src:     `package main; func main() {}`,
					},
					{
						RelPath: "testContextDir/Dockerfile",
						Src: `FROM alpine:3.5
RUN echo 'Product: \{\{Product\}\}'
RUN echo 'Version: \{\{Version\}\}'
RUN echo 'Repository: \{\{Repository\}\}'
`,
					},
				},
				ConfigFiles: map[string]string{
					"godel/config/godel.yml": godelYML,
					"godel/config/dist-plugin.yml": `
products:
  foo:
    build:
      main-pkg: ./foo
    dist:
      disters:
        type: os-arch-bin
    docker:
      docker-builders:
        tester:
          type: default
          context-dir: testContextDir
          tag-templates:
            release: tester-tag:{{Version}}
            snapshot: tester-tag:snapshot
`,
				},
				Args: []string{
					"build",
					"--dry-run",
					"--tags=release",
				},
				WantOutput: func(projectDir string) string {
					return fmt.Sprintf(`[DRY RUN] Running Docker build for configuration tester of product foo...
[DRY RUN] Run [docker build --file %s/testContextDir/Dockerfile -t tester-tag:1.0.0 %s/testContextDir]
`, projectDir, projectDir)
				},
			},
		},
	)
}

func TestUpgradeConfig(t *testing.T) {
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
    docker:
      - repository: foo/foo
        tag: snapshot
        context-dir: ./build/docker
        build-args-script: |
          echo "--no-cache"
`,
				},
				Legacy: true,
				WantOutput: `Upgraded configuration for dist-plugin.yml
`,
				WantFiles: map[string]string{
					"godel/config/dist-plugin.yml": `products:
  foo:
    build: {}
    docker:
      docker-builders:
        docker-image-0:
          type: default
          script: |
            #!/bin/bash
            echo "--no-cache"
          context-dir: ./build/docker
          tag-templates:
            default: '{{Repository}}foo/foo:snapshot'
`,
				},
			},
			{
				Name: `valid v0 config with tags specified as array works`,
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
        type: os-arch-bin
        config:
          os-archs:
            - os: darwin
              arch: amd64
            - os: linux
              arch: amd64
    docker:
      docker-builders:
        tester:
          type: default
          config:
            build-args:
              # comment
              - "--rm"
          context-dir: testContextDir
          tag-templates:
            - tester-tag:latest-and-greatest
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
        type: os-arch-bin
        config:
          os-archs:
            - os: darwin
              arch: amd64
            - os: linux
              arch: amd64
    docker:
      docker-builders:
        tester:
          type: default
          config:
            build-args:
              # comment
              - "--rm"
          context-dir: testContextDir
          tag-templates:
            - tester-tag:latest-and-greatest
`,
				},
			},
			{
				Name: `valid v0 config with tags specified as map works and maintains ordering`,
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
        type: os-arch-bin
        config:
          os-archs:
            - os: darwin
              arch: amd64
            - os: linux
              arch: amd64
    docker:
      docker-builders:
        tester:
          type: default
          config:
            build-args:
              # comment
              - "--rm"
          context-dir: testContextDir
          tag-templates:
            "4": tester-tag:latest-and-greatest
            "2": tester-tag:1
            "1": tester-tag:3
            "3": tester-tag:2
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
        type: os-arch-bin
        config:
          os-archs:
            - os: darwin
              arch: amd64
            - os: linux
              arch: amd64
    docker:
      docker-builders:
        tester:
          type: default
          config:
            build-args:
              # comment
              - "--rm"
          context-dir: testContextDir
          tag-templates:
            "4": tester-tag:latest-and-greatest
            "2": tester-tag:1
            "1": tester-tag:3
            "3": tester-tag:2
`,
				},
			},
		},
	)
}
