package migrations

import "imove-server/server/sqlstore/migrator"

// --- Migration Guide line ---
// 1. Never change a migration that is committed and pushed to master
// 2. Always add new migrations (to change or undo previous migrations)
// 3. Some migrations are not yet written (rename column, table, drop table, index etc)

func AddMigrations(mg *migrator.Migrator) {
	addMigrationLogMigrations(mg)
}

func addMigrationLogMigrations(mg *migrator.Migrator) {
	migrationLogV1 := migrator.Table{
		Name: "migration_log",
		Columns: []*migrator.Column{
			{Name: "id", Type: migrator.DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "migration_id", Type: migrator.DB_NVarchar, Length: 255},
			{Name: "sql", Type: migrator.DB_Text},
			{Name: "success", Type: migrator.DB_Bool},
			{Name: "error", Type: migrator.DB_Text},
			{Name: "timestamp", Type: migrator.DB_DateTime},
		},
	}

	mg.AddMigration("create migration_log table", migrator.NewAddTableMigration(migrationLogV1))
}
