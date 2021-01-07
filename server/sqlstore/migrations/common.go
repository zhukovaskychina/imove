package migrations

import (
	"fmt"
	"imove-server/server/sqlstore/migrator"
)

func addDropAllIndicesMigrations(mg *migrator.Migrator, versionSuffix string, table migrator.Table) {
	for _, index := range table.Indices {
		migrationId := fmt.Sprintf("drop index %s - %s", index.XName(table.Name), versionSuffix)
		mg.AddMigration(migrationId, migrator.NewDropIndexMigration(table, index))
	}
}

//
func addTableIndicesMigrations(mg *migrator.Migrator, versionSuffix string, table migrator.Table) {
	for _, index := range table.Indices {
		migrationId := fmt.Sprintf("create index %s - %s", index.XName(table.Name), versionSuffix)
		mg.AddMigration(migrationId, migrator.NewAddIndexMigration(table, index))
	}
}

//
func addTableRenameMigration(mg *migrator.Migrator, oldName string, newName string, versionSuffix string) {
	migrationId := fmt.Sprintf("Rename table %s to %s - %s", oldName, newName, versionSuffix)
	mg.AddMigration(migrationId, migrator.NewRenameTableMigration(oldName, newName))
}

//
func addTableReplaceMigrations(mg *migrator.Migrator, from migrator.Table, to migrator.Table, migrationVersion int64, tableDataMigration map[string]string) {
	fromV := version(migrationVersion - 1)
	toV := version(migrationVersion)
	tmpTableName := to.Name + "_tmp_qwerty"

	createTable := fmt.Sprintf("create %v %v", to.Name, toV)
	copyTableData := fmt.Sprintf("copy %v %v to %v", to.Name, fromV, toV)
	dropTable := fmt.Sprintf("drop %v", tmpTableName)

	addDropAllIndicesMigrations(mg, fromV, from)
	addTableRenameMigration(mg, from.Name, tmpTableName, fromV)
	mg.AddMigration(createTable, migrator.NewAddTableMigration(to))
	addTableIndicesMigrations(mg, toV, to)
	mg.AddMigration(copyTableData, migrator.NewCopyTableDataMigration(to.Name, tmpTableName, tableDataMigration))
	mg.AddMigration(dropTable, migrator.NewDropTableMigration(tmpTableName))
}

//
func version(v int64) string {
	return fmt.Sprintf("v%v", v)
}
