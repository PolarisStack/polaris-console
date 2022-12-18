/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package mysql

import (
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/polarismesh/polaris-console/common/log"
	"github.com/polarismesh/polaris-console/store"
)

const (
	// SystemNamespace system namespace
	SystemNamespace = "Polaris"
	// STORENAME database storage name
	STORENAME = "defaultStore"
	// DefaultConnMaxLifetime default maximum connection lifetime
	DefaultConnMaxLifetime = 60 * 30 // 默认是30分钟
)

// init 自动引入包初始化函数
func init() {
	s := &stableStore{}
	_ = store.RegisterStore(s)
}

// stableStore 实现了Store接口
type stableStore struct {
	*lockStore
	*alarmRuleStore

	// 主数据库，可以进行读写
	master *BaseDB
	// 对主数据库的事务操作，可读写
	masterTx *BaseDB
	// 备数据库，提供只读
	slave *BaseDB
	start bool
}

// Name 实现Name函数
func (s *stableStore) Name() string {
	return STORENAME
}

// Initialize 初始化函数
func (s *stableStore) Initialize(conf *store.Config) error {
	if s.start {
		return nil
	}

	masterConfig, slaveConfig, err := parseDatabaseConf(conf.Option)
	if err != nil {
		return err
	}
	master, err := NewBaseDB(masterConfig)
	if err != nil {
		return err
	}
	s.master = master

	masterTx, err := NewBaseDB(masterConfig)
	if err != nil {
		return err
	}
	s.masterTx = masterTx

	if slaveConfig != nil {
		log.Infof("[Store][database] use slave database config: %+v", slaveConfig)
		slave, err := NewBaseDB(slaveConfig)
		if err != nil {
			return err
		}
		s.slave = slave
	}
	// 如果slave为空，意味着slaveConfig为空，用master数据库替代
	if s.slave == nil {
		s.slave = s.master
	}

	log.Infof("[Store][database] connect the database successfully")

	s.start = true
	s.newStore()
	return nil
}

// parseDatabaseConf return slave, master, error
func parseDatabaseConf(opt map[string]interface{}) (*dbConfig, *dbConfig, error) {
	// 必填
	masterEnter, ok := opt["master"]
	if !ok || masterEnter == nil {
		return nil, nil, errors.New("database master db config is missing")
	}
	masterConfig, err := parseStoreConfig(masterEnter)
	if err != nil {
		return nil, nil, err
	}

	// 只读数据库可选
	slaveEntry, ok := opt["slave"]
	if !ok || slaveEntry == nil {
		return masterConfig, nil, nil
	}
	slaveConfig, err := parseStoreConfig(slaveEntry)
	if err != nil {
		return nil, nil, err
	}

	return masterConfig, slaveConfig, nil
}

// parseStoreConfig 解析store的配置
func parseStoreConfig(opts interface{}) (*dbConfig, error) {
	obj, _ := opts.(map[interface{}]interface{})

	needCheckFields := map[string]string{"dbType": "", "dbUser": "", "dbPwd": "", "dbAddr": "", "dbName": ""}

	for key := range needCheckFields {
		val, ok := obj[key]
		if !ok {
			return nil, fmt.Errorf("config Plugin %s:%s type must be string", STORENAME, key)
		}

		needCheckFields[key] = fmt.Sprintf("%v", val)
	}

	c := &dbConfig{
		dbType: needCheckFields["dbType"],
		dbUser: needCheckFields["dbUser"],
		dbPwd:  needCheckFields["dbPwd"],
		dbAddr: needCheckFields["dbAddr"],
		dbName: needCheckFields["dbName"],
	}
	if maxOpenConns, _ := obj["maxOpenConns"].(int); maxOpenConns > 0 {
		c.maxOpenConns = maxOpenConns
	}
	if maxIdleConns, _ := obj["maxIdleConns"].(int); maxIdleConns > 0 {
		c.maxIdleConns = maxIdleConns
	}
	c.connMaxLifetime = DefaultConnMaxLifetime
	if connMaxLifetime, _ := obj["connMaxLifetime"].(int); connMaxLifetime > 0 {
		c.connMaxLifetime = connMaxLifetime
	}

	if isolationLevel, _ := obj["txIsolationLevel"].(int); isolationLevel > 0 {
		c.txIsolationLevel = isolationLevel
	}
	return c, nil
}

// Destroy 退出函数
func (s *stableStore) Destroy() error {
	s.start = false
	if s.master != nil {
		_ = s.master.Close()
	}
	if s.masterTx != nil {
		_ = s.masterTx.Close()
	}
	if s.slave != nil {
		_ = s.slave.Close()
	}
	s.master = nil
	s.masterTx = nil
	s.slave = nil

	return nil
}

func (s *stableStore) newStore() {
	s.lockStore = &lockStore{master: s.master, slave: s.slave}
	s.alarmRuleStore = &alarmRuleStore{master: s.master, slave: s.slave}
}