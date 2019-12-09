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
	"math/rand"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
	"github.com/rs/xid"
	"go.uber.org/zap"

	"cellery.io/cellery-observability/components/global/observability-agent/pkg/store"
)

type (
	Persister struct {
		logger    *zap.SugaredLogger
		directory string
	}
	Transaction struct {
		Lock *flock.Flock
	}
	File struct {
		Path string `json:"path"`
	}
)

func (transaction *Transaction) Commit() error {
	err := os.Remove(transaction.Lock.String())
	if err != nil {
		return fmt.Errorf("could not delete the published file : %v", err)
	}
	return nil
}

func (transaction *Transaction) Rollback() error {
	err := transaction.Lock.Unlock()
	if err != nil {
		return fmt.Errorf("could not unlock the file")
	}
	return nil
}

func (persister *Persister) Write(str string) error {
	fileLock := persister.createFile()
	persister.logger.Debugf("Created a new file : %s", fileLock.String())
	locked, err := fileLock.TryLock()
	if err != nil {
		return fmt.Errorf("could not lock the created file : %v", err)
	}
	if !locked {
		return fmt.Errorf("could not lock the created file")
	}
	defer persister.unlock(fileLock)

	bytesArr := []byte(str)
	err = ioutil.WriteFile(fileLock.String(), bytesArr, 0644)
	if err != nil {
		return fmt.Errorf("could not write to the file : %v", err)
	}
	return nil
}

func (persister *Persister) createFile() *flock.Flock {
	uuid := xid.New().String()
	fileLock := flock.New(fmt.Sprintf("%s/%s.json", persister.directory, uuid))
	return fileLock
}

func (persister *Persister) unlock(flock *flock.Flock) {
	err := flock.Unlock()
	if err != nil {
		persister.logger.Warn("Could not unlock the file")
	}
}

func (persister *Persister) Fetch() (string, store.Transaction, error) {
	files, err := filepath.Glob(persister.directory + "/*.json")
	if err != nil {
		return "", &Transaction{}, fmt.Errorf("could not read the given directory %s : %v", persister.directory,
			err)
	}
	persister.logger.Debugf("Files in the directory : %s", files)
	if len(files) > 0 {
		transaction := &Transaction{
			Lock: flock.New(files[rand.Intn(len(files))]),
		}
		return persister.read(transaction)
	} else {
		return "", &Transaction{}, nil
	}
}

func (persister *Persister) read(transaction *Transaction) (string, *Transaction, error) {
	locked, err := transaction.Lock.TryLock()
	if err != nil {
		return "", transaction, fmt.Errorf("could not lock the file : %v", err)
	}
	if !locked {
		return "", transaction, fmt.Errorf("could not achieve the lock")
	}

	data, err := ioutil.ReadFile(transaction.Lock.String())
	if err != nil {
		return "", transaction, fmt.Errorf("could not read the file : %v", err)
	}
	if data == nil || string(data) == "" {
		err = os.Remove(transaction.Lock.String())
		persister.logger.Debugf("Could not remove the empty file : %v", err)
		return "", transaction, fmt.Errorf("file is empty, hence removed")
	}
	return string(data), transaction, nil
}

func NewPersister(config *File, logger *zap.SugaredLogger) (*Persister, error) {
	path := config.Path
	ps := &Persister{
		logger:    logger,
		directory: path,
	}
	_, err := os.Stat(path)
	if err == nil {
		return ps, nil
	}
	if os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("could not make the directory : %v", err)
		}
		return ps, nil
	} else {
		return nil, fmt.Errorf("error when checking the existance of the file path : %v", err)
	}
}
