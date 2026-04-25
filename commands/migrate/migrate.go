// Copyright 2013 bee authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package migrate

import (
	"os"

	"container/list"
	"fmt"
	"log"

	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/models"
)

var (
	migrationList = &migrationCache{}
)

type MigrationDatabase interface {
	//获取当前的版本
	Version() int64
	//校验当前是否可更新
	ValidUpdate(version int64) error
	//校验并备份表结构
	ValidForBackupTableSchema() error
	//校验并更新表结构
	ValidForUpdateTableSchema() error
	//恢复旧数据
	MigrationOldTableData() error
	//插入新数据
	MigrationNewTableData() error
	//增加迁移记录
	AddMigrationRecord(version int64) error
	//最后的清理工作
	MigrationCleanup() error
	//回滚本次迁移
	RollbackMigration() error
}

type migrationCache struct {
	items *list.List
}

func RunMigration() {

	if len(os.Args) >= 2 && os.Args[1] == "migrate" {

		migrate, err := models.NewMigration().FindFirst()

		if err != nil {
			//log.Fatalf("migrations table %s", err)
			migrate = models.NewMigration()
		}
		fmt.Println("Start migration databae... ")

		for el := migrationList.items.Front(); el != nil; el = el.Next() {

			//如果存在比当前版本大的版本，则依次升级
			if item, ok := el.Value.(MigrationDatabase); ok && item.Version() > migrate.Version {
				err := item.ValidUpdate(migrate.Version)
				if err != nil {
					log.Fatal(err)
				}
				err = item.ValidForBackupTableSchema()
				if err != nil {
					item.RollbackMigration()
					log.Fatal(err)
				}
				err = item.ValidForUpdateTableSchema()
				if err != nil {
					item.RollbackMigration()
					log.Fatal(err)
				}
				err = item.MigrationOldTableData()
				if err != nil {
					item.RollbackMigration()
					log.Fatal(err)
				}
				err = item.MigrationNewTableData()
				if err != nil {
					item.RollbackMigration()
					log.Fatal(err)
				}
				err = item.AddMigrationRecord(item.Version())
				if err != nil {
					item.RollbackMigration()
					log.Fatal(err)
				}
				err = item.MigrationCleanup()
				if err != nil {
					item.RollbackMigration()
					log.Fatal(err)
				}
			}
		}
		fmt.Println("Migration successfull.")
		os.Exit(0)
	}
}

//导出数据库的表结构
func ExportDatabaseTable() ([]string, error) {
	dbadapter, _ := conf.GetString("db_adapter")
	dbdatabase, _ := conf.GetString("db_database")
	tables := make([]string, 0)

	db := models.GetDB()
	switch dbadapter {
	case "mysql":
		{
			type tableName struct {
				TableName string
			}
			var lists []tableName

			if err := db.Raw(fmt.Sprintf("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '%s'", dbdatabase)).Scan(&lists).Error; err != nil {
				return tables, err
			}
			for _, t := range lists {
				type createTableResult struct {
					CreateTable string
				}
				var results []createTableResult
				if err := db.Raw(fmt.Sprintf("show create table %s", t.TableName)).Scan(&results).Error; err != nil {
					return tables, err
				}
				if len(results) > 0 {
					tables = append(tables, results[0].CreateTable)
				}
			}
			break
		}
	case "sqlite3":
		{
			type sqliteMaster struct {
				SQL string
			}
			var results []sqliteMaster
			if err := db.Raw("SELECT sql FROM sqlite_master WHERE sql IS NOT NULL ORDER BY rootpage ASC").Scan(&results).Error; err != nil {
				return tables, err
			}
			for _, item := range results {
				tables = append(tables, item.SQL)
			}
			break
		}

	}
	return tables, nil
}

func RegisterMigration() {
	migrationList.items = list.New()

	migrationList.items.PushBack(NewMigrationVersion03())
}
