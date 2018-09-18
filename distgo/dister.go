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

package distgo

type Dister interface {
	// TypeName returns the type of this dister.
	TypeName() (string, error)

	// Artifacts returns the names of the artifacts generated by running RunDist.
	Artifacts(renderedName string) ([]string, error)

	// PackagingExtension returns the extension of the primary artifact generated by this dister. May return an empty
	// string if the dister does not have a notion of a primary artifact or if it does not have an extension. Used
	// primarily for generating the POM for disters that produce Maven-style artifacts -- the value returned by this
	// function should be able to be used as the "packaging" property in a POM. The returned extension should not
	// include a leading period: for example, a valid value may be "tar.gz" or "zip".
	PackagingExtension() (string, error)

	// RunDist runs the distribution task. The provided DistID specifies the ID for the current dister and the
	// ProductTaskOutputInfo contains all of the output information for the product. When this function is called, the
	// DistWorkDir for the product should have already been created and the dist script should have already been run.
	// Running this function should populate the DistWorkDir with any files and/or directories required for the
	// distribution. However, the dist artifacts themselves should not be created. May return a value that is the
	// JSON-serialized bytes that is provided as the runDistResult parameter to GenerateDistArtifacts.
	RunDist(distID DistID, productTaskOutputInfo ProductTaskOutputInfo) ([]byte, error)

	// GenerateDistArtifacts generates the dist artifact outputs from the DistWorkDir for the distribution. The
	// runDistResult argument contains the JSON-serialized bytes returned by the RunDist call. The DistWorkDir contains
	// the output generated by RunDist. The outputs written by GenerateDistArtifacts should match the artifacts returned
	// by the Artifacts function.
	GenerateDistArtifacts(distID DistID, productTaskOutputInfo ProductTaskOutputInfo, runDistResult []byte) error
}

type DisterFactory interface {
	Types() []string
	NewDister(typeName string, cfgYMLBytes []byte) (Dister, error)
	ConfigUpgrader(typeName string) (ConfigUpgrader, error)
}
