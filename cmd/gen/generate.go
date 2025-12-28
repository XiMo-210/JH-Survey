package main

import (
	"github.com/spf13/cobra"
	"github.com/zjutjh/mygo/foundation/command"
	"github.com/zjutjh/mygo/ndb"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"

	"app/register"
)

var tables = []string{
	"admin",
	"survey",
	"result",
	"stats",
}

func main() {
	command.Execute(
		register.Boot,
		func(c *cobra.Command) {},
		func(cmd *cobra.Command, args []string) error { return nil },
	)

	g := gen.NewGenerator(gen.Config{
		OutPath: "./dao/query",
		Mode:    gen.WithDefaultQuery | gen.WithQueryInterface,
	})
	g.UseDB(ndb.Pick())

	m := map[string]func(columnType gorm.ColumnType) (dataType string){
		"tinyint": func(columnType gorm.ColumnType) (dataType string) {
			return "int8"
		},
	}
	g.WithDataTypeMap(m)

	for _, table := range tables {
		opts := []gen.ModelOpt{
			gen.FieldType("deleted_at", "soft_delete.DeletedAt"),
			gen.FieldGORMTag("deleted_at", func(tag field.GormTag) field.GormTag {
				return tag.Set("softDelete", "milli")
			}),
			gen.FieldJSONTag("deleted_at", "-"),
		}

		var model any
		if table == "stats" {
			model = g.GenerateModelAs(table, "Stats", opts...)
		} else {
			model = g.GenerateModel(table, opts...)
		}

		g.ApplyBasic(model)
	}

	g.Execute()
}
