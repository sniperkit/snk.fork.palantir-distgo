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

package legacy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/palantir/godel/pkg/osarch"
	"github.com/palantir/godel/pkg/versionedconfig"
	"github.com/palantir/pkg/matcher"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/palantir/distgo/dister/bin"
	"github.com/palantir/distgo/dister/manual"
	"github.com/palantir/distgo/dister/osarchbin"
	"github.com/palantir/distgo/distgo"
	"github.com/palantir/distgo/distgo/config/internal/v0"
	"github.com/palantir/distgo/dockerbuilder/defaultdockerbuilder"
)

type Project struct {
	versionedconfig.ConfigWithLegacy `yaml:",inline,omitempty"`

	// Products maps product names to configurations.
	Products map[string]Product `yaml:"products,omitempty"`

	// BuildOutputDir specifies the default build output directory for products executables built by the "build"
	// command. The executables generated by "build" will be written to this directory unless the location is
	// overridden by the product-specific configuration.
	BuildOutputDir string `yaml:"build-output-dir,omitempty"`

	// DistOutputDir specifies the default distribution output directory for product distributions created by the
	// "dist" command. The distribution directory and artifact generated by "dist" will be written to this directory
	// unless the location is overridden by the product-specific configuration.
	DistOutputDir string `yaml:"dist-output-dir,omitempty"`

	// DistScriptInclude is script content that is prepended to any non-empty ProductDistCfg.Script. It can be used
	// to define common functionality used in the distribution script for multiple different products.
	DistScriptInclude string `yaml:"dist-script-include,omitempty"`

	// GroupID is the identifier used as the group ID for the POM.
	GroupID string `yaml:"group-id,omitempty"`

	// Exclude matches the paths to exclude when determining the projects to build.
	Exclude matcher.NamesPathsCfg `yaml:"exclude,omitempty"`
}

type Product struct {
	// Build specifies the build configuration for the product.
	Build Build `yaml:"build,omitempty"`

	// Run specifies the run configuration for the product.
	Run Run `yaml:"run,omitempty"`

	// Dist specifies the dist configurations for the product.
	Dist RawDistConfigs `yaml:"dist,omitempty"`

	// DockerImages specifies the docker build configurations for the product.
	DockerImages []DockerImage `yaml:"docker,omitempty"`

	// DefaultPublish specifies the publish configuration that is applied to distributions that do not specify their
	// own publish configurations.
	DefaultPublish Publish `yaml:"publish,omitempty"`
}

type Build struct {
	// Skip specifies whether the build step should be skipped entirely. Its primary use is for products that handle
	// their own build logic in the "dist" step ("dist-only" products).
	Skip bool `yaml:"skip,omitempty"`

	// Script is the content of a script that is written to file a file and run before this product is built. The
	// contents of this value are written to a file with a header `#!/bin/bash` and executed. The script process
	// inherits the environment variables of the Go process and also has the following environment variables
	// defined:
	//
	//   PROJECT_DIR: the root directory of project
	//   PRODUCT: product name
	//   VERSION: product version
	//   IS_SNAPSHOT: 1 if the version contains a git hash as part of the string, 0 otherwise
	Script string `yaml:"script,omitempty"`

	// MainPkg is the location of the main package for the product relative to the root directory. For example,
	// "./distgo/main".
	MainPkg string `yaml:"main-pkg,omitempty"`

	// OutputDir is the directory to which the executable is written.
	OutputDir string `yaml:"output-dir,omitempty"`

	// BuildArgsScript is the content of a script that is written to a file and run before this product is built
	// to provide supplemental build arguments for the product. The contents of this value are written to a file
	// with a header `#!/bin/bash` and executed. The script process inherits the environment variables of the Go
	// process. Each line of output of the script is provided to the "build" command as a separate argument. For
	// example, the following script would add the arguments "-ldflags" "-X" "main.year=$YEAR" to the build command:
	//
	//   build-args-script: |
	//                      YEAR=$(date +%Y)
	//                      echo "-ldflags"
	//                      echo "-X"
	//                      echo "main.year=$YEAR"
	BuildArgsScript string `yaml:"build-args-script,omitempty"`

	// VersionVar is the path to a variable that is set with the version information for the build. For example,
	// "github.com/palantir/godel/cmd/godel.Version". If specified, it is provided to the "build" command as an
	// ldflag.
	VersionVar string `yaml:"version-var,omitempty"`

	// Environment specifies values for the environment variables that should be set for the build. For example,
	// the following sets CGO to false:
	//
	//   environment:
	//     CGO_ENABLED: "0"
	Environment map[string]string `yaml:"environment,omitempty"`

	// OSArchs specifies the GOOS and GOARCH pairs for which the product is built. If blank, defaults to the GOOS
	// and GOARCH of the host system at runtime.
	OSArchs []osarch.OSArch `yaml:"os-archs,omitempty"`
}

type Run struct {
	// Args contain the arguments provided to the product when invoked using the "run" task.
	Args []string `yaml:"args,omitempty"`
}

type Dist struct {
	// OutputDir is the directory to which the distribution is written.
	OutputDir string `yaml:"output-dir,omitempty"`

	// Path is the path (from the project root) to a directory whose contents will be copied into the output
	// distribution directory at the beginning of the "dist" command. Can be used to include static resources and
	// other files required in a distribution.
	InputDir string `yaml:"input-dir,omitempty"`

	// InputProducts is a slice of the names of products in the project (other than the current one) whose binaries
	// are required for the "dist" task. The "dist" task will ensure that the outputs of "build" exist for all of
	// the products specified in this slice (and will build the products as part of the task if necessary) and make
	// the outputs available to the "dist" script as environment variables. Note that the "dist" task only
	// guarantees that the products will be built and their locations will be available in the environment variables
	// provided to the script -- it is the responsibility of the user to write logic in the dist script to copy the
	// generated binaries.
	InputProducts []string `yaml:"input-products,omitempty"`

	// Script is the content of a script that is written to file a file and run after the initial distribution
	// process but before the artifact generation process. The contents of this value are written to a file with a
	// header `#!/bin/bash` with the contents of the global `dist-script-include` prepended and executed. The script
	// process inherits the environment variables of the Go process and also has the following environment variables
	// defined:
	//
	//   DIST_DIR: the absolute path to the root directory of the distribution created for the current product
	//   PROJECT_DIR: the root directory of project
	//   PRODUCT: product name
	//   VERSION: product version
	//   IS_SNAPSHOT: 1 if the version contains a git hash as part of the string, 0 otherwise
	Script string `yaml:"script,omitempty"`

	// DistType specifies the type of the distribution to be built and configuration for it. If unspecified,
	// defaults to a DistInfo of type OSArchBinDistType.
	DistType DistInfo `yaml:"dist-type,omitempty"`

	// Publish is the configuration for the "publish" task.
	Publish Publish `yaml:"publish,omitempty"`
}

type DistInfo struct {
	// Type is the type of the distribution. Value should be a valid value defined by DistInfoType.
	Type string `yaml:"type,omitempty"`

	// Info is the configuration content of the dist info.
	Info interface{} `yaml:"info,omitempty"`
}

type DistInfoType string

const (
	SLSDistType       DistInfoType = "sls"         // distribution that uses the Standard Layout Specification
	BinDistType       DistInfoType = "bin"         // distribution that includes all of the binaries for a product
	RPMDistType       DistInfoType = "rpm"         // RPM distribution
	OSArchBinDistType DistInfoType = "os-arch-bin" // distribution that consists of the binaries for a specific OS/Architecture
	ManualDistType    DistInfoType = "manual"      // distribution that consists of a distribution whose output is created by the distribution script
)

type BinDist struct {
	versionedconfig.ConfigWithLegacy `yaml:",inline,omitempty"`

	// OmitInitSh specifies whether or not the distribution should omit the auto-generated initialization script for the
	// product (a script in the "bin" directory that chooses the binary to invoke based on the host platform). If the
	// value is present and false, then the initialization script will be generated; otherwise, it will not.
	OmitInitSh *bool `yaml:"omit-init-sh,omitempty"`
	// InitShTemplateFile is the relative path to the template that should be used to generate the "init.sh" script.
	// If the value is absent, the default template will be used.
	InitShTemplateFile string `yaml:"init-sh-template-file,omitempty"`
}

type ManualDist struct {
	versionedconfig.ConfigWithLegacy `yaml:",inline,omitempty"`

	// Extension is the extension used by the target output generated by the dist script: for example, "tgz",
	// "zip", etc. Extension is used to locate the output generated by the dist script. The output should be a file
	// of the form "{{product-name}}-{{version}}.{{Extension}}". If Extension is empty, it is assumed that the
	// output has no extension and is of the form "{{product-name}}-{{version}}".
	Extension string `yaml:"extension,omitempty"`
}

type OSArchBinDist struct {
	versionedconfig.ConfigWithLegacy `yaml:",inline,omitempty"`

	// OSArchs specifies the GOOS and GOARCH pairs for which TGZ distributions are created. If blank, defaults to
	// the GOOS and GOARCH of the host system at runtime.
	OSArchs []osarch.OSArch `yaml:"os-archs,omitempty"`
}

type SLSDist struct {
	versionedconfig.ConfigWithLegacy `yaml:",inline,omitempty"`

	// InitShTemplateFile is the path to a template file that is used as the basis for the init.sh script of the
	// distribution. The path is relative to the project root directory. The contents of the file is processed using
	// Go templates and is provided with a distgo.ProductBuildSpec struct. If omitted, the default init.sh script
	// is used.
	InitShTemplateFile string `yaml:"init-sh-template-file,omitempty"`

	// ManifestTemplateFile is the path to a template file that is used as the basis for the manifest.yml file of
	// the distribution. The path is relative to the project root directory. The contents of the file is processed
	// using Go templates and is provided with a distgo.ProductBuildSpec struct.
	ManifestTemplateFile string `yaml:"manifest-template-file,omitempty"`

	// ServiceArgs is the string provided as the service arguments for the default init.sh file generated for the distribution.
	ServiceArgs string `yaml:"service-args,omitempty"`

	// ProductType is the SLS product type for the distribution.
	ProductType string `yaml:"product-type,omitempty"`

	// ManifestExtensions contain the SLS manifest extensions for the distribution.
	ManifestExtensions map[string]interface{} `yaml:"manifest-extensions,omitempty"`

	// Reloadable will enable the `init.sh reload` command which sends SIGHUP to the process.
	Reloadable bool `yaml:"reloadable,omitempty"`

	// YMLValidationExclude specifies a matcher used to specify YML files or paths that should not be validated as
	// part of creating the distribution. By default, the SLS distribution task verifies that all "*.yml" and
	// "*.yaml" files in the distribution are syntactically valid. If a distribution is known to ship with YML files
	// that are not valid YML, this parameter can be used to exclude those files from validation.
	YMLValidationExclude matcher.NamesPathsCfg `yaml:"yml-validation-exclude,omitempty"`
}

type RPMDist struct {
	versionedconfig.ConfigWithLegacy `yaml:",inline,omitempty"`

	// Release is the release identifier that forms part of the name/version/release/architecture quadruplet
	// uniquely identifying the RPM package. Default is "1".
	Release string `yaml:"release,omitempty"`

	// ConfigFiles is a slice of absolute paths within the RPM that correspond to configuration files. RPM
	// identifies these as mutable. Default is no files.
	ConfigFiles []string `yaml:"config-files,omitempty"`

	// BeforeInstallScript is the content of shell script to run before this RPM is installed. Optional.
	BeforeInstallScript string `yaml:"before-install-script,omitempty"`

	// AfterInstallScript is the content of shell script to run immediately after this RPM is installed. Optional.
	AfterInstallScript string `yaml:"after-install-script,omitempty"`

	// AfterRemoveScript is the content of shell script to clean up after this RPM is removed. Optional.
	AfterRemoveScript string `yaml:"after-remove-script,omitempty"`
}

type DockerDep struct {
	Product    string `yaml:"product,omitempty"`
	Type       string `yaml:"type,omitempty"`
	TargetFile string `yaml:"target-file,omitempty"`
}

type DockerImage struct {
	Repository      string          `yaml:"repository,omitempty"`
	Tag             string          `yaml:"tag,omitempty"`
	ContextDir      string          `yaml:"context-dir,omitempty"`
	Deps            []DockerDep     `yaml:"dependencies,omitempty"`
	Info            DockerImageInfo `yaml:"info,omitempty"`
	BuildArgsScript string          `yaml:"build-args-script,omitempty"`
}

type DockerImageInfo struct {
	Type string      `yaml:"type,omitempty"`
	Data interface{} `yaml:"data,omitempty"`
}

type SLSDockerImageInfo struct {
	Legacy bool `yaml:"legacy-config,omitempty"`

	GroupID      string                 `yaml:"group-id,omitempty"`
	ProuductType string                 `yaml:"product-type,omitempty"`
	Extensions   map[string]interface{} `yaml:"manifest-extensions,omitempty"`
}

type Publish struct {
	// GroupID is the product-specific configuration equivalent to the global GroupID configuration.
	GroupID string `yaml:"group-id,omitempty"`

	// Almanac contains the parameters for Almanac publish operations. Optional.
	Almanac Almanac `yaml:"almanac,omitempty"`
}

type Almanac struct {
	versionedconfig.ConfigWithLegacy `yaml:",inline"`

	// Metadata contains the metadata provided to the Almanac publish task.
	Metadata map[string]string `yaml:"metadata"`

	// Tags contains the tags provided to the Almanac publish task.
	Tags []string `yaml:"tags"`
}

type RawDistConfigs []Dist

func (out *RawDistConfigs) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var multiple []Dist
	if err := unmarshal(&multiple); err == nil {
		if len(multiple) == 0 {
			return errors.New("if `dist` key is specified, there must be at least one dist")
		}
		*out = multiple
		return nil
	}

	var single Dist
	if err := unmarshal(&single); err != nil {
		// return the error from a single DistConfig if neither one works
		return err
	}
	*out = []Dist{single}
	return nil
}

func (cfg *DistInfo) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// unmarshal type alias (uses default unmarshal strategy)
	type rawDistInfoConfigAlias DistInfo
	var rawAliasConfig rawDistInfoConfigAlias
	if err := unmarshal(&rawAliasConfig); err != nil {
		return err
	}

	rawDistInfoConfig := DistInfo(rawAliasConfig)
	switch DistInfoType(rawDistInfoConfig.Type) {
	case SLSDistType:
		type typedRawConfig struct {
			Type string
			Info SLSDist
		}
		var rawSLS typedRawConfig
		if err := unmarshal(&rawSLS); err != nil {
			return err
		}
		rawDistInfoConfig.Info = rawSLS.Info
	case BinDistType:
		type typedRawConfig struct {
			Type string
			Info BinDist
		}
		var rawBin typedRawConfig
		if err := unmarshal(&rawBin); err != nil {
			return err
		}
		rawDistInfoConfig.Info = rawBin.Info
	case ManualDistType:
		type typedRawConfig struct {
			Type string
			Info ManualDist
		}
		var rawManual typedRawConfig
		if err := unmarshal(&rawManual); err != nil {
			return err
		}
		rawDistInfoConfig.Info = rawManual.Info
	case OSArchBinDistType:
		type typedRawConfig struct {
			Type string
			Info OSArchBinDist
		}
		var rawOSArchBin typedRawConfig
		if err := unmarshal(&rawOSArchBin); err != nil {
			return err
		}
		rawDistInfoConfig.Info = rawOSArchBin.Info
	case RPMDistType:
		type typedRawConfig struct {
			Type string
			Info RPMDist
		}
		var rawRPM typedRawConfig
		if err := unmarshal(&rawRPM); err != nil {
			return err
		}
		rawDistInfoConfig.Info = rawRPM.Info
	}
	*cfg = rawDistInfoConfig
	return nil
}

func (cfg *DockerImageInfo) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// unmarshal type alias (uses default unmarshal strategy)
	type rawImageInfoConfigAlias DockerImageInfo
	var rawAliasConfig rawImageInfoConfigAlias
	if err := unmarshal(&rawAliasConfig); err != nil {
		return err
	}

	rawImageInfoConfig := DockerImageInfo(rawAliasConfig)
	switch rawImageInfoConfig.Type {
	case "sls":
		type typedRawConfig struct {
			Type string
			Data SLSDockerImageInfo
		}
		var rawSLS typedRawConfig
		if err := unmarshal(&rawSLS); err != nil {
			return err
		}
		rawImageInfoConfig.Data = rawSLS.Data
	}
	*cfg = rawImageInfoConfig
	return nil
}

func UpgradeConfig(
	cfgBytes []byte,
	disterFactory distgo.DisterFactory,
	dockerBuilderFactory distgo.DockerBuilderFactory,
	publisherFactory distgo.PublisherFactory) ([]byte, error) {

	var legacyCfg Project
	if err := yaml.UnmarshalStrict(cfgBytes, &legacyCfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal dist-plugin legacy configuration")
	}

	upgradedCfg, err := upgradeLegacyConfig(legacyCfg, disterFactory, dockerBuilderFactory, publisherFactory)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// indicates that this is the default config
	if upgradedCfg == nil {
		return nil, nil
	}

	outputBytes, err := yaml.Marshal(*upgradedCfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal dist-plugin v0 configuration")
	}
	return outputBytes, nil
}

func upgradeLegacyConfig(
	legacyCfg Project,
	disterFactory distgo.DisterFactory,
	dockerBuilderFactory distgo.DockerBuilderFactory,
	publisherFactory distgo.PublisherFactory) (*v0.ProjectConfig, error) {

	// upgrade top-level configuration
	upgradedCfg := v0.ProjectConfig{}

	var productDefaults v0.ProductConfig
	// BuildOutputDir
	if legacyCfg.BuildOutputDir != "" {
		productDefaults.Build = &v0.BuildConfig{
			OutputDir: stringPtr(legacyCfg.BuildOutputDir),
		}
	}
	// DistOutputDir
	if legacyCfg.DistOutputDir != "" {
		productDefaults.Dist = &v0.DistConfig{
			OutputDir: stringPtr(legacyCfg.DistOutputDir),
		}
	}
	// DistScriptInclude
	upgradedCfg.ScriptIncludes = translateEnvVars(legacyCfg.DistScriptInclude)
	// GroupID
	if legacyCfg.GroupID != "" {
		productDefaults.Publish = &v0.PublishConfig{
			GroupID: stringPtr(legacyCfg.GroupID),
		}
	}
	upgradedCfg.ProductDefaults = productDefaults

	// Exclude
	upgradedCfg.Exclude = legacyCfg.Exclude

	// Products
	var sortedProductsKeys []string
	for k := range legacyCfg.Products {
		sortedProductsKeys = append(sortedProductsKeys, k)
	}
	sort.Strings(sortedProductsKeys)

	var products map[distgo.ProductID]v0.ProductConfig
	if len(sortedProductsKeys) > 0 {
		products = make(map[distgo.ProductID]v0.ProductConfig, len(sortedProductsKeys))
	}

	for _, k := range sortedProductsKeys {
		legacyProduct := legacyCfg.Products[k]
		var product v0.ProductConfig
		if !legacyProduct.Build.Skip {
			product.Build = &v0.BuildConfig{}
			// Script
			if legacyProduct.Build.Script != "" {
				return nil, errors.Errorf(`dist-plugin legacy configuration sets "script" field for the build configuration for %s, but build scripts are no longer supported`, k)
			}
			// MainPkg
			if legacyProduct.Build.MainPkg != "" {
				product.Build.MainPkg = stringPtr(legacyProduct.Build.MainPkg)
			}
			// OutputDir
			if legacyProduct.Build.OutputDir != "" {
				product.Build.OutputDir = stringPtr(legacyProduct.Build.OutputDir)
			}
			// BuildArgsScript
			if legacyProduct.Build.BuildArgsScript != "" {
				product.Build.BuildArgsScript = stringPtr(legacyProduct.Build.BuildArgsScript)
			}
			// VersionVar
			if legacyProduct.Build.VersionVar != "" {
				product.Build.VersionVar = stringPtr(legacyProduct.Build.VersionVar)
			}
			// Environment
			if len(legacyProduct.Build.Environment) > 0 {
				envMap := legacyProduct.Build.Environment
				product.Build.Environment = &envMap
			}
			// OSArchs
			if len(legacyProduct.Build.OSArchs) > 0 {
				osArchs := legacyProduct.Build.OSArchs
				product.Build.OSArchs = &osArchs
			}
		}
		if len(legacyProduct.Run.Args) > 0 {
			// Args
			runArgs := legacyProduct.Run.Args
			product.Run = &v0.RunConfig{
				Args: &runArgs,
			}
		}

		if len(legacyProduct.Dist) == 0 && product.Build.OSArchs != nil && len(*product.Build.OSArchs) > 0 {
			// if previous configuration did not specify a "dist" configuration but has build OS/Arch combinations,
			// previous behavior was to default to os-arch-bin dist type, so explicitly add that. Note that the behavior
			// where a blank os/arch would build an os-arch-bin dist for the current OS is no longer supported.
			legacyProduct.Dist = RawDistConfigs{
				{
					DistType: DistInfo{
						Type: string(OSArchBinDistType),
						Info: OSArchBinDist{
							ConfigWithLegacy: versionedconfig.ConfigWithLegacy{
								Legacy: true,
							},
							OSArchs: *product.Build.OSArchs,
						},
					},
				},
			}
		}

		if len(legacyProduct.Dist) > 0 {
			if len(legacyProduct.Dist) > 1 {
				return nil, errors.Errorf(`product %q specifies more than 1 dist configuration and migrations are not supported for these configurations`, k)
			}
			legacyDist := legacyProduct.Dist[0]

			var upgradedDisterCfg v0.DisterConfig
			var legacyDisterCfgBytes []byte
			distType := DistInfoType(legacyDist.DistType.Type)
			switch distType {
			case SLSDistType:
				upgradedDisterCfg.Type = stringPtr("sls")

				slsDist := legacyDist.DistType.Info.(SLSDist)
				slsDist.Legacy = true
				cfgBytes, err := yaml.Marshal(slsDist)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to marshal legacy SLS dist configuration for %q", k)
				}
				legacyDisterCfgBytes = cfgBytes
			case BinDistType:
				upgradedDisterCfg.Type = stringPtr(bin.TypeName)

				// upgrade legacy bin configuration manually into script rather than upgrading into current bin config
				binDist := legacyDist.DistType.Info.(BinDist)
				if binDist.OmitInitSh != nil && *binDist.OmitInitSh == false {
					if binDist.InitShTemplateFile != "" {
						return nil, errors.Errorf("bin dist for product %q specifies a custom init.sh template file, which is no longer supported", k)
					}
					if upgradedDisterCfg.Script == nil {
						upgradedDisterCfg.Script = stringPtr("")
					}
					upgradedDisterCfg.Script = stringPtr(appendToScript(*upgradedDisterCfg.Script, binDistInitShScript()))
				}
			case OSArchBinDistType:
				upgradedDisterCfg.Type = stringPtr(osarchbin.TypeName)

				osArchBinDist := legacyDist.DistType.Info.(OSArchBinDist)
				osArchBinDist.Legacy = true
				cfgBytes, err := yaml.Marshal(osArchBinDist)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to marshal legacy os-arch-bin dist configuration for %q", k)
				}
				legacyDisterCfgBytes = cfgBytes
			case ManualDistType:
				upgradedDisterCfg.Type = stringPtr(manual.TypeName)

				manualDist := legacyDist.DistType.Info.(ManualDist)
				manualDist.Legacy = true
				cfgBytes, err := yaml.Marshal(manualDist)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to marshal legacy manual dist configuration for %q", k)
				}
				legacyDisterCfgBytes = cfgBytes
			case RPMDistType:
				upgradedDisterCfg.Type = stringPtr("rpm")

				rpmDist := legacyDist.DistType.Info.(RPMDist)
				rpmDist.Legacy = true
				cfgBytes, err := yaml.Marshal(rpmDist)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to marshal legacy rpm dist configuration for %q", k)
				}
				legacyDisterCfgBytes = cfgBytes
			default:
				return nil, errors.Errorf(`product %q specifies unsupported dist type %q`, k, distType)
			}

			if len(legacyDisterCfgBytes) > 0 {
				upgrader, err := disterFactory.ConfigUpgrader(string(distType))
				if err != nil {
					return nil, errors.Wrapf(err, "no upgrader found for legacy dister for product %q", k)
				}
				upgradedBytes, err := upgrader.UpgradeConfig(legacyDisterCfgBytes)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to upgrade legacy configuration for dister for product %q", k)
				}
				var cfgMapSlice yaml.MapSlice
				if err := yaml.UnmarshalStrict(upgradedBytes, &cfgMapSlice); err != nil {
					return nil, errors.Wrapf(err, "failed to unmarshal upgraded dist configuration for %q as yaml.MapSlice", k)
				}
				upgradedDisterCfg.Config = &cfgMapSlice
			}

			product.Dist = &v0.DistConfig{}
			// OutputDir
			if legacyDist.OutputDir != "" {
				product.Dist.OutputDir = stringPtr(legacyDist.OutputDir)
			}
			// InputDir
			if legacyDist.InputDir != "" {
				if upgradedDisterCfg.Script == nil {
					upgradedDisterCfg.Script = stringPtr("")
				}
				upgradedDisterCfg.Script = stringPtr(appendToScript(*upgradedDisterCfg.Script, inputDirScriptContent(legacyDist.InputDir)))
			}
			// InputProducts
			if len(legacyDist.InputProducts) > 0 {
				var deps []distgo.ProductID
				for _, product := range legacyDist.InputProducts {
					deps = addDepIfNotPresent(deps, distgo.ProductID(product))
				}
				product.Dependencies = &deps
			}
			// Script
			if legacyDist.Script != "" {
				if upgradedDisterCfg.Script == nil {
					upgradedDisterCfg.Script = stringPtr("")
				}
				upgradedDisterCfg.Script = stringPtr(appendToScript(*upgradedDisterCfg.Script, translateEnvVars(legacyDist.Script)))
			}
			// DistType
			product.Dist.Disters = &v0.DistersConfig{
				distgo.DistID(*upgradedDisterCfg.Type): upgradedDisterCfg,
			}

			// Publish
			var publishCfg *v0.PublishConfig
			if legacyDist.Publish.GroupID != "" {
				if publishCfg == nil {
					publishCfg = &v0.PublishConfig{}
				}
				publishCfg.GroupID = stringPtr(legacyDist.Publish.GroupID)
			}
			if len(legacyDist.Publish.Almanac.Tags) > 0 || len(legacyDist.Publish.Almanac.Metadata) > 0 {
				if publishCfg == nil {
					publishCfg = &v0.PublishConfig{}
				}
				if publishCfg.PublishInfo == nil {
					publishCfg.PublishInfo = &map[distgo.PublisherTypeID]v0.PublisherConfig{}
				}

				legacyDist.Publish.Almanac.Legacy = true
				legacyPublisherBytes, err := yaml.Marshal(legacyDist.Publish.Almanac)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to marshal legacy Almanac publish configuration for product %q", k)
				}
				upgrader, err := publisherFactory.ConfigUpgrader("artifactory-almanac")
				if err != nil {
					return nil, errors.Wrapf(err, "no upgrader found for legacy publisher for product %q", k)
				}
				upgradedBytes, err := upgrader.UpgradeConfig(legacyPublisherBytes)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to upgrade legacy configuration for publisher for product %q", k)
				}
				var cfgMapSlice yaml.MapSlice
				if err := yaml.UnmarshalStrict(upgradedBytes, &cfgMapSlice); err != nil {
					return nil, errors.Wrapf(err, "failed to unmarshal upgraded publisher configuration for %q as yaml.MapSlice", k)
				}
				(*publishCfg.PublishInfo)[distgo.PublisherTypeID("artifactory-almanac")] = v0.PublisherConfig{
					Config: &cfgMapSlice,
				}
			}
			product.Publish = publishCfg
		}

		if len(legacyProduct.DockerImages) > 0 {
			upgradedProductDockerConfig := v0.DockerConfig{}
			upgradedDockerBuildersConfig := make(v0.DockerBuildersConfig)
			upgradedProductDockerConfig.DockerBuildersConfig = &upgradedDockerBuildersConfig

			for i, legacyDockerImage := range legacyProduct.DockerImages {
				upgradedDockerBuilderConfig := v0.DockerBuilderConfig{}

				// Tag
				if legacyDockerImage.Tag != "" {
					upgradedDockerBuilderConfig.TagTemplates = &[]string{
						legacyDockerImage.Tag,
					}
				}
				// ContextDir
				if legacyDockerImage.ContextDir != "" {
					upgradedDockerBuilderConfig.ContextDir = stringPtr(legacyDockerImage.ContextDir)
				}
				// Deps
				if len(legacyDockerImage.Deps) > 0 {
					for _, currLegacyDockerDep := range legacyDockerImage.Deps {
						switch dockerDepType := currLegacyDockerDep.Type; dockerDepType {
						case "docker":
							// nothing to do here (add to product's dependency list after switch)
						case "bin", "sls":
							// add as a dist dependency
							if upgradedDockerBuilderConfig.InputDists == nil {
								upgradedDockerBuilderConfig.InputDists = &[]distgo.ProductDistID{}
							}
							inputDistID := distgo.ProductDistID(fmt.Sprintf("%s.%s", currLegacyDockerDep.Product, dockerDepType))
							inputDistIDPresent := false
							for _, currID := range *upgradedDockerBuilderConfig.InputDists {
								if inputDistID == currID {
									inputDistIDPresent = true
									break
								}
							}
							if !inputDistIDPresent {
								*upgradedDockerBuilderConfig.InputDists = append(*upgradedDockerBuilderConfig.InputDists, inputDistID)
							}
						default:
							return nil, errors.Errorf("product %q has a Docker image configuration that declares a Docker dependency of type %q, which is not supported", k, dockerDepType)
						}

						// add to product's dependency list
						if product.Dependencies == nil {
							product.Dependencies = &[]distgo.ProductID{}
						}

						currLegacyDockerDepProductID := distgo.ProductID(currLegacyDockerDep.Product)
						if currLegacyDockerDepProductID != distgo.ProductID(k) {
							*product.Dependencies = addDepIfNotPresent(*product.Dependencies, distgo.ProductID(currLegacyDockerDep.Product))
						}
					}
				}

				if legacyDockerImage.Info.Type == "sls" {
					upgradedDockerBuilderConfig.Type = stringPtr("sls")

					dockerImageInfo := legacyDockerImage.Info.Data.(SLSDockerImageInfo)
					dockerImageInfo.Legacy = true
					cfgBytes, err := yaml.Marshal(dockerImageInfo)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to marshal legacy sls docker builder configuration for %q", k)
					}
					legacyDockerBuilderCfgBytes := cfgBytes
					if len(legacyDockerBuilderCfgBytes) > 0 {
						upgrader, err := dockerBuilderFactory.ConfigUpgrader(legacyDockerImage.Info.Type)
						if err != nil {
							return nil, errors.Wrapf(err, "no upgrader found for legacy dockerbuilder for product %q", k)
						}
						upgradedBytes, err := upgrader.UpgradeConfig(legacyDockerBuilderCfgBytes)
						if err != nil {
							return nil, errors.Wrapf(err, "failed to upgrade legacy configuration for dockerbuilder for product %q", k)
						}
						var cfgMapSlice yaml.MapSlice
						if err := yaml.UnmarshalStrict(upgradedBytes, &cfgMapSlice); err != nil {
							return nil, errors.Wrapf(err, "failed to unmarshal upgraded dockerbuilder configuration for %q as yaml.MapSlice", k)
						}
						upgradedDockerBuilderConfig.Config = &cfgMapSlice
					}
				} else {
					// use default builder
					upgradedDockerBuilderConfig.Type = stringPtr(defaultdockerbuilder.TypeName)
				}

				dockerID := distgo.DockerID(fmt.Sprintf("docker-image-%d", i))
				upgradedDockerBuildersConfig[dockerID] = upgradedDockerBuilderConfig
			}
			product.Docker = &upgradedProductDockerConfig
		}

		// if new configuration has defaults for "publish" operation and current product does not have any publish
		// configuration, give it an empty publish configuration so the defaults are used.
		if upgradedCfg.ProductDefaults.Publish != nil && product.Publish == nil {
			product.Publish = &v0.PublishConfig{}
		}

		products[distgo.ProductID(k)] = product
	}
	upgradedCfg.Products = products

	return &upgradedCfg, nil
}

func stringPtr(in string) *string {
	return &in
}

// appendToScript appends the provided string to the given script. If the given script is empty, it prepends it with the
// line `#!/bin/bash`. Ensures that the resulting script ends in a newline (which makes it easier to append subsequent
// elements if needed).
func appendToScript(script, append string) string {
	if script == "" {
		script += `#!/bin/bash
`
	}
	script += append
	if !strings.HasSuffix(script, "\n") {
		script += "\n"
	}
	return script
}

func inputDirScriptContent(inputDir string) string {
	return `### START: auto-generated back-compat code for "input-dir" behavior ###
` + fmt.Sprintf(`cp -r "$PROJECT_DIR"/%s/. "$DIST_WORK_DIR"
find "$DIST_WORK_DIR" -type f -name .gitkeep -exec rm '{}' \;`, inputDir) + `
### END: auto-generated back-compat code for "input-dir" behavior ###`
}

func binDistInitShScript() string {
	return `### START: auto-generated back-compat code for "omit-init-sh: false" behavior for bin dist ###
read -d '' GODELUPGRADED_scriptContent <<"EOF"
#!/bin/bash
set -euo pipefail
BIN_DIR="$(cd "$(dirname "$0")" && pwd)"
# determine OS
OS=""
case "$(uname)" in
  Darwin*)
    OS=darwin
    ;;
  Linux*)
    OS=linux
    ;;
  *)
    echo "Unsupported operating system: $(uname)"
    exit 1
    ;;
esac
# determine executable location based on OS
CMD=$BIN_DIR/$OS-amd64/{{.ProductName}}
# verify that executable exists
if [ ! -e "$CMD" ]; then
    echo "Executable $CMD does not exist"
    exit 1
fi
# invoke appropriate executable
$CMD "$@"
EOF
GODELUPGRADED_templated=${GODELUPGRADED_scriptContent//\{\{.ProductName\}\}/$PRODUCT}
echo "$GODELUPGRADED_templated" > "$DIST_WORK_DIR"/bin/"$PRODUCT".sh
chmod 755 "$DIST_WORK_DIR"/bin/"$PRODUCT".sh
### END: auto-generated back-compat code for "omit-init-sh: false" behavior for bin dist ###`
}

func translateEnvVars(in string) string {
	out := in
	if strings.Contains(out, "IS_SNAPSHOT") {
		const isSnapshotScript = `### START: auto-generated back-compat code for "IS_SNAPSHOT" variable ###
IS_SNAPSHOT=0
if [[ $VERSION =~ .+g[-+.]?[a-fA-F0-9]{3,}$ ]]; then IS_SNAPSHOT=1; fi
### END: auto-generated back-compat code for "IS_SNAPSHOT" variable ###
`
		out = isSnapshotScript + out
	}
	out = translateVar(out, "DIST_DIR", "DIST_WORK_DIR")
	return out
}

func translateVar(in, oldVar, newVar string) string {
	out := in
	out = strings.Replace(out, fmt.Sprintf("$%s", oldVar), fmt.Sprintf("$%s", newVar), -1)
	out = strings.Replace(out, fmt.Sprintf("${%s}", oldVar), fmt.Sprintf("${%s}", newVar), -1)
	return out
}

func addDepIfNotPresent(deps []distgo.ProductID, productID distgo.ProductID) []distgo.ProductID {
	productIDPresent := false
	for _, currID := range deps {
		if productID == currID {
			productIDPresent = true
			break
		}
	}
	if productIDPresent {
		return deps
	}
	return append(deps, productID)
}
