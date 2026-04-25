package migrate

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mindoc-org/mindoc/models"
)

type MigrationVersion03 struct {
	isValid bool
	tables  []string
}

func NewMigrationVersion03() *MigrationVersion03 {
	return &MigrationVersion03{isValid: false, tables: make([]string, 0)}
}

func (m *MigrationVersion03) Version() int64 {
	return 201705271114
}

func (m *MigrationVersion03) ValidUpdate(version int64) error {
	if m.Version() > version {
		m.isValid = true
		return nil
	}
	m.isValid = false
	return errors.New("The target version is higher than the current version.")
}

func (m *MigrationVersion03) ValidForBackupTableSchema() error {
	if !m.isValid {
		return errors.New("The current version failed to verify.")
	}
	var err error
	m.tables, err = ExportDatabaseTable()

	return err
}

func (m *MigrationVersion03) ValidForUpdateTableSchema() error {
	if !m.isValid {
		return errors.New("The current version failed to verify.")
	}

	err := models.GetDB().AutoMigrate(
		&models.Member{}, &models.Book{}, &models.Relationship{},
		&models.Option{}, &models.Document{}, &models.Attachment{},
		&models.Logger{}, &models.MemberToken{}, &models.DocumentHistory{},
		&models.Migration{}, &models.Label{}, &models.Blog{},
		&models.Template{}, &models.Team{}, &models.TeamMember{},
		&models.TeamRelationship{}, &models.Itemsets{}, &models.Comment{},
		&models.CommentVote{}, &models.ContentReverseIndex{},
		&models.WorkWeixinAccount{}, &models.DingTalkAccount{},
	)

	if err != nil {
		return err
	}

	//_,err = o.Raw("ALTER TABLE md_members ADD auth_method VARCHAR(50) DEFAULT 'local' NULL").Exec()

	return err
}

func (m *MigrationVersion03) MigrationOldTableData() error {
	if !m.isValid {
		return errors.New("The current version failed to verify.")
	}
	return nil
}

func (m *MigrationVersion03) MigrationNewTableData() error {
	if !m.isValid {
		return errors.New("The current version failed to verify.")
	}
	db := models.GetDB()

	if err := db.Exec("UPDATE md_members SET auth_method = 'local'").Error; err != nil {
		return err
	}
	if err := db.Exec("INSERT INTO md_options (option_title, option_name, option_value) SELECT '是否启用文档历史','ENABLE_DOCUMENT_HISTORY','true' WHERE NOT exists(SELECT * FROM md_options WHERE option_name = 'ENABLE_DOCUMENT_HISTORY')").Error; err != nil {
		return err
	}
	return nil
}

func (m *MigrationVersion03) AddMigrationRecord(version int64) error {
	db := models.GetDB()
	tables, err := ExportDatabaseTable()

	if err != nil {
		return err
	}
	migration := models.NewMigration()
	migration.Version = version
	migration.Status = "update"
	migration.CreateTime = time.Now()
	migration.Name = fmt.Sprintf("update_%d", version)
	migration.Statements = strings.Join(tables, "\r\n")

	return db.Create(migration).Error
}

func (m *MigrationVersion03) MigrationCleanup() error {

	return nil
}

func (m *MigrationVersion03) RollbackMigration() error {
	if !m.isValid {
		return errors.New("The current version failed to verify.")
	}
	db := models.GetDB()
	if err := db.Exec("ALTER TABLE md_members DROP COLUMN auth_method").Error; err != nil {
		return err
	}

	if err := db.Exec("DROP TABLE md_document_history").Error; err != nil {
		return err
	}
	if err := db.Exec("DELETE FROM md_options WHERE option_name = 'ENABLE_DOCUMENT_HISTORY'").Error; err != nil {
		return err
	}

	return nil
}
