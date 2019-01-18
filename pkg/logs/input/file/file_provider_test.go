// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

// +build !windows

package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/DataDog/datadog-agent/pkg/logs/types"
	"github.com/stretchr/testify/suite"

	"github.com/DataDog/datadog-agent/pkg/logs/config"
)

type ProviderTestSuite struct {
	suite.Suite
	testDir    string
	filesLimit int
}

// newLogSources returns a new log source initialized with the right path.
func (suite *ProviderTestSuite) newLogSources(path string) []*config.LogSource {
	return []*config.LogSource{config.NewLogSource("", &config.LogsConfig{Type: types.FileType, Path: path})}
}

func (suite *ProviderTestSuite) SetupTest() {
	suite.filesLimit = 3

	// Create temporary directory
	var err error
	suite.testDir, err = ioutil.TempDir("", "log-file_provider-test-")
	suite.Nil(err)

	// Create directory tree:
	path := fmt.Sprintf("%s/1", suite.testDir)
	err = os.Mkdir(path, os.ModePerm)
	suite.Nil(err)

	path = fmt.Sprintf("%s/1/1.log", suite.testDir)
	_, err = os.Create(path)
	suite.Nil(err)

	path = fmt.Sprintf("%s/1/2.log", suite.testDir)
	_, err = os.Create(path)
	suite.Nil(err)

	path = fmt.Sprintf("%s/1/3.log", suite.testDir)
	_, err = os.Create(path)
	suite.Nil(err)

	path = fmt.Sprintf("%s/2", suite.testDir)
	err = os.Mkdir(path, os.ModePerm)
	suite.Nil(err)

	path = fmt.Sprintf("%s/2/1.log", suite.testDir)
	_, err = os.Create(path)
	suite.Nil(err)

	path = fmt.Sprintf("%s/2/2.log", suite.testDir)
	_, err = os.Create(path)
	suite.Nil(err)
}

func (suite *ProviderTestSuite) TearDownTest() {
	os.Remove(suite.testDir)
}

func (suite *ProviderTestSuite) TestFilesToTailReturnsSpecificFile() {
	path := fmt.Sprintf("%s/1/1.log", suite.testDir)
	fileProvider := NewProvider(suite.filesLimit)
	logSources := suite.newLogSources(path)
	files := fileProvider.FilesToTail(logSources)

	suite.Equal(1, len(files))
	suite.Equal(fmt.Sprintf("%s/1/1.log", suite.testDir), files[0].Path)
	suite.Equal(make([]string, 0), logSources[0].Messages.GetMessages())
	suite.Equal(make([]string, 0), logSources[0].Messages.GetWarnings())
}

func (suite *ProviderTestSuite) TestFilesToTailReturnsAllFilesFromDirectory() {
	path := fmt.Sprintf("%s/1/*.log", suite.testDir)
	fileProvider := NewProvider(suite.filesLimit)
	logSources := suite.newLogSources(path)
	files := fileProvider.FilesToTail(logSources)

	suite.Equal(3, len(files))
	suite.Equal(fmt.Sprintf("%s/1/1.log", suite.testDir), files[0].Path)
	suite.Equal(fmt.Sprintf("%s/1/2.log", suite.testDir), files[1].Path)
	suite.Equal(fmt.Sprintf("%s/1/3.log", suite.testDir), files[2].Path)
	suite.Equal([]string{"3 files tailed out of 3 files matching"}, logSources[0].Messages.GetMessages())
	suite.Equal(
		[]string{
			"The limit on the maximum number of files in use (3) has been reached. Increase this limit (thanks to the attribute logs_config.open_files_limit in datadog.yaml) or decrease the number of tailed file.",
		},
		logSources[0].Messages.GetWarnings(),
	)
}

func (suite *ProviderTestSuite) TestFilesToTailReturnsAllFilesFromAnyDirectoryWithRightPermissions() {
	path := fmt.Sprintf("%s/*/*1.log", suite.testDir)
	fileProvider := NewProvider(suite.filesLimit)
	logSources := suite.newLogSources(path)
	files := fileProvider.FilesToTail(logSources)

	suite.Equal(2, len(files))
	suite.Equal(fmt.Sprintf("%s/1/1.log", suite.testDir), files[0].Path)
	suite.Equal(fmt.Sprintf("%s/2/1.log", suite.testDir), files[1].Path)
	suite.Equal([]string{"2 files tailed out of 2 files matching"}, logSources[0].Messages.GetMessages())
	suite.Equal(make([]string, 0), logSources[0].Messages.GetWarnings())
}

func (suite *ProviderTestSuite) TestFilesToTailReturnsSpecificFileWithWildcard() {
	path := fmt.Sprintf("%s/1/?.log", suite.testDir)
	fileProvider := NewProvider(suite.filesLimit)
	logSources := suite.newLogSources(path)
	files := fileProvider.FilesToTail(logSources)

	suite.Equal(3, len(files))
	suite.Equal(fmt.Sprintf("%s/1/1.log", suite.testDir), files[0].Path)
	suite.Equal(fmt.Sprintf("%s/1/2.log", suite.testDir), files[1].Path)
	suite.Equal(fmt.Sprintf("%s/1/3.log", suite.testDir), files[2].Path)
	suite.Equal([]string{"3 files tailed out of 3 files matching"}, logSources[0].Messages.GetMessages())
	suite.Equal(
		[]string{
			"The limit on the maximum number of files in use (3) has been reached. Increase this limit (thanks to the attribute logs_config.open_files_limit in datadog.yaml) or decrease the number of tailed file.",
		},
		logSources[0].Messages.GetWarnings(),
	)
}

func (suite *ProviderTestSuite) TestNumberOfFilesToTailDoesNotExceedLimit() {
	path := fmt.Sprintf("%s/*/*.log", suite.testDir)
	fileProvider := NewProvider(suite.filesLimit)
	logSources := suite.newLogSources(path)
	files := fileProvider.FilesToTail(logSources)
	suite.Equal(suite.filesLimit, len(files))
	suite.Equal([]string{"3 files tailed out of 5 files matching"}, logSources[0].Messages.GetMessages())
	suite.Equal(
		[]string{
			"The limit on the maximum number of files in use (3) has been reached. Increase this limit (thanks to the attribute logs_config.open_files_limit in datadog.yaml) or decrease the number of tailed file.",
		},
		logSources[0].Messages.GetWarnings(),
	)
}

func (suite *ProviderTestSuite) TestAllWildcardPathsAreUpdated() {
	filesLimit := 2
	fileProvider := NewProvider(filesLimit)
	logSources := []*config.LogSource{
		config.NewLogSource("", &config.LogsConfig{Type: types.FileType, Path: fmt.Sprintf("%s/1/*.log", suite.testDir)}),
		config.NewLogSource("", &config.LogsConfig{Type: types.FileType, Path: fmt.Sprintf("%s/2/*.log", suite.testDir)}),
	}
	files := fileProvider.FilesToTail(logSources)
	suite.Equal(2, len(files))
	suite.Equal([]string{"2 files tailed out of 3 files matching"}, logSources[0].Messages.GetMessages())
	suite.Equal(
		[]string{
			"The limit on the maximum number of files in use (2) has been reached. Increase this limit (thanks to the attribute logs_config.open_files_limit in datadog.yaml) or decrease the number of tailed file.",
		},
		logSources[0].Messages.GetWarnings(),
	)
	suite.Equal([]string{"0 files tailed out of 2 files matching"}, logSources[1].Messages.GetMessages())
	suite.Equal(
		[]string{
			"The limit on the maximum number of files in use (2) has been reached. Increase this limit (thanks to the attribute logs_config.open_files_limit in datadog.yaml) or decrease the number of tailed file.",
		},
		logSources[1].Messages.GetWarnings(),
	)

	os.Remove(fmt.Sprintf("%s/1/2.log", suite.testDir))
	os.Remove(fmt.Sprintf("%s/1/3.log", suite.testDir))
	os.Remove(fmt.Sprintf("%s/2/2.log", suite.testDir))
	files = fileProvider.FilesToTail(logSources)
	suite.Equal(2, len(files))
	suite.Equal([]string{"1 files tailed out of 1 files matching"}, logSources[0].Messages.GetMessages())
	suite.Equal(make([]string, 0), logSources[0].Messages.GetWarnings())

	suite.Equal([]string{"1 files tailed out of 1 files matching"}, logSources[1].Messages.GetMessages())
	suite.Equal(
		[]string{
			"The limit on the maximum number of files in use (2) has been reached. Increase this limit (thanks to the attribute logs_config.open_files_limit in datadog.yaml) or decrease the number of tailed file.",
		},
		logSources[1].Messages.GetWarnings(),
	)

	os.Remove(fmt.Sprintf("%s/2/1.log", suite.testDir))

	files = fileProvider.FilesToTail(logSources)
	suite.Equal(1, len(files))
	suite.Equal([]string{"1 files tailed out of 1 files matching"}, logSources[0].Messages.GetMessages())
	suite.Equal(make([]string, 0), logSources[0].Messages.GetWarnings())

	suite.Equal([]string{"0 files tailed out of 0 files matching"}, logSources[1].Messages.GetMessages())
	suite.Equal(make([]string, 0), logSources[0].Messages.GetWarnings())
}

func TestProviderTestSuite(t *testing.T) {
	suite.Run(t, new(ProviderTestSuite))
}
