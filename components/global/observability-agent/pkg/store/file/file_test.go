/*
 * Copyright (c) 2019, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 *
 * WSO2 Inc. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofrs/flock"

	"cellery.io/cellery-observability/components/global/observability-agent/pkg/logging"
)

var (
	testStr = "{\"contextReporterKind\":\"inbound\", \"destinationUID\":\"kubernetes://istio-policy-74d6c8b4d5-mmr49.istio-system\", \"requestID\":\"6e544e82-2a0c-4b83-abcc-0f62b89cdf3f\", \"requestMethod\":\"POST\", \"requestPath\":\"/istio.mixer.v1.Mixer/Check\", \"requestTotalSize\":\"2748\", \"responseCode\":\"200\", \"responseDurationNanoSec\":\"695653\", \"responseTotalSize\":\"199\", \"sourceUID\":\"kubernetes://pet-be--controller-deployment-6f6f5768dc-n9jf7.default\", \"spanID\":\"ae295f3a4bbbe537\", \"traceID\":\"b55a0f7f20d36e49f8612bac4311791d\"}"
)

func TestWriteWithoutErrors(t *testing.T) {
	logger, err := logging.NewLogger()
	if err != nil {
		t.Errorf("Error building logger: %v", err)
	}
	persister := &Persister{
		logger:    logger,
		directory: ".",
	}
	err = persister.Write(fmt.Sprintf("[%s]", testStr))
	if err != nil {
		t.Error("Could not write")
	}
	files, err := filepath.Glob("./*.json")
	for _, fname := range files {
		err = os.Remove(fname)
	}
}

func TestWriteWithInvalidDirectory(t *testing.T) {
	logger, err := logging.NewLogger()
	if err != nil {
		t.Errorf("Error building logger: %v", err)
	}
	persister := &Persister{
		logger:    logger,
		directory: "./wrong_directory",
	}
	err = persister.Write(fmt.Sprintf("[%s]", testStr))
	expectedErr := "no such file or directory"
	if err == nil {
		t.Errorf("An error was not thrown, but expected : %s", expectedErr)
		return
	}
	if strings.Contains(expectedErr, err.Error()) {
		t.Errorf("Expected error was not thrown, received error : %v", err)
	}
	files, err := filepath.Glob("./*.json")
	for _, fname := range files {
		err = os.Remove(fname)
	}
}

func TestFetchWithoutErrors(t *testing.T) {
	logger, err := logging.NewLogger()
	if err != nil {
		t.Errorf("Error building logger: %v", err)
	}
	persister := &Persister{
		logger:    logger,
		directory: "./",
	}
	_ = ioutil.WriteFile("./test.json", []byte(testStr), 0644)
	str, _, _ := persister.Fetch()
	if str != testStr {
		t.Error("Contents are not equal")
	}
	files, err := filepath.Glob("./*.json")
	for _, fname := range files {
		err = os.Remove(fname)
	}
}

func TestFetchWithEmptyFile(t *testing.T) {
	logger, err := logging.NewLogger()
	if err != nil {
		t.Errorf("Error building logger: %v", err)
	}
	persister := &Persister{
		logger:    logger,
		directory: "./",
	}
	_ = ioutil.WriteFile("./test.json", []byte(""), 0644)
	_, _, err = persister.Fetch()
	expectedErr := "file is empty, hence removed"
	if err == nil {
		t.Errorf("An error was not thrown, but expected : %s", expectedErr)
		return
	}
	if err.Error() != expectedErr {
		t.Errorf("Expected error was not thrown, received error : %v", err)
	}
	files, err := filepath.Glob("./*.json")
	for _, fname := range files {
		err = os.Remove(fname)
	}
}

func TestFetchWithInvalidDirectory(t *testing.T) {
	logger, err := logging.NewLogger()
	if err != nil {
		t.Errorf("Error building logger: %v", err)
	}
	persister := &Persister{
		logger:    logger,
		directory: "./wrong_dir",
	}
	str, _, err := persister.Fetch()
	if str != "" {
		t.Error("Unexpected behaviour from the function")
	}
}

func TestCommitWithoutErrors(t *testing.T) {
	_ = ioutil.WriteFile("./test.json", []byte(testStr), 0644)
	transaction := &Transaction{
		Lock: flock.New("./test.json"),
	}
	err := transaction.Commit()
	if err != nil {
		t.Errorf("Error when committing : %v", err)
	}
	files, err := filepath.Glob("./*.json")
	for _, fname := range files {
		err = os.Remove(fname)
	}
}

func TestCommitWithError(t *testing.T) {
	transaction := &Transaction{
		Lock: flock.New("./test.json"),
	}
	err := transaction.Commit()
	expectedErr := "could not delete the published file : remove ./test.json: no such file or directory"
	if err == nil {
		t.Errorf("An error was not thrown, but expected : %s", expectedErr)
		return
	}
	if err.Error() != expectedErr {
		t.Errorf("Expected error was not thrown, received error : %v", err)
	}
}

func TestRollback(t *testing.T) {
	_ = ioutil.WriteFile("./test.json", []byte(testStr), 0644)
	transaction := &Transaction{
		Lock: flock.New("./test.json"),
	}
	_, _ = transaction.Lock.TryLock()
	err := transaction.Rollback()
	if err != nil {
		t.Errorf("An error was thrown when unlocking the file : %v", err)
	}
	files, err := filepath.Glob("./*.json")
	for _, fname := range files {
		err = os.Remove(fname)
	}
}

func TestNewMethodWithoutDir(t *testing.T) {
	config := &File{Path: "./testDir"}
	logger, err := logging.NewLogger()
	if err != nil {
		t.Errorf("Error building logger: %v", err)
	}
	_, _ = NewPersister(config, logger)
	_, err = os.Stat(config.Path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Error("Directory has not been created")
		} else {
			t.Errorf("An unexpected error has been occured : %v", err)
		}
	}
	_ = os.RemoveAll(config.Path)
}

func TestNewMethodWithDir(t *testing.T) {
	config := &File{Path: "./testDir"}
	err := os.MkdirAll(config.Path, os.ModePerm)
	if err != nil {
		t.Errorf("error occurred when creating the directory : %v", err)
	}
	logger, err := logging.NewLogger()
	if err != nil {
		t.Errorf("Error building logger: %v", err)
	}
	_, err = NewPersister(config, logger)
	if err != nil {
		t.Errorf("An unexpected error has been occured : %v", err)
	}
	_ = os.RemoveAll(config.Path)
}
