package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/droundy/goopt"
	"github.com/gobuffalo/packr/v2"
	"github.com/jimsmart/schema"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/lib/pq"
	"github.com/logrusorgru/aurora"
	_ "github.com/mattn/go-sqlite3"
	"github.com/opannapo/structof/src/dbmeta"
)

var (
	sqlType          = goopt.String([]string{"--sqltype"}, "mysql", "sql database type such as [ mysql, mssql, postgres, sqlite, etc. ]")
	sqlConnStr       = goopt.String([]string{"-c", "--connstr"}, "nil", "database connection string")
	sqlDatabase      = goopt.String([]string{"-d", "--database"}, "nil", "Database to for connection")
	sqlTable         = goopt.String([]string{"-t", "--table"}, "", "Table to build struct from")
	excludeSQLTables = goopt.String([]string{"-x", "--exclude"}, "", "Table(s) to exclude")
	templateDir      = goopt.String([]string{"--templateDir"}, "", "Template Dir")

	modelPackageName    = goopt.String([]string{"--model"}, "model", "name to set for model package")
	modelNamingTemplate = goopt.String([]string{"--model_naming"}, "{{FmtFieldName .}}", "model naming template to name structs")
	fieldNamingTemplate = goopt.String([]string{"--field_naming"}, "{{FmtFieldName (stringifyFirstChar .) }}", "field naming template to name structs")
	fileNamingTemplate  = goopt.String([]string{"--file_naming"}, "{{.}}", "file_naming template to name files")

	outDir    = goopt.String([]string{"--out"}, ".", "output dir")
	overwrite = goopt.Flag([]string{"--overwrite"}, []string{"--no-overwrite"}, "Overwrite existing files (default)", "disable overwriting files")

	addJSONAnnotation = goopt.Flag([]string{"--json"}, []string{"--no-json"}, "Add json annotations (default)", "Disable json annotations")
	jsonNameFormat    = goopt.String([]string{"--json-fmt"}, "snake", "json name format [snake | camel | lower_camel | none]")

	addGormAnnotation     = goopt.Flag([]string{"--gorm"}, []string{}, "Add gorm annotations (tags)", "")
	addProtobufAnnotation = goopt.Flag([]string{"--protobuf"}, []string{}, "Add protobuf annotations (tags)", "")
	protoNameFormat       = goopt.String([]string{"--proto-fmt"}, "snake", "proto name format [snake | camel | lower_camel | none]")
	gogoProtoImport       = goopt.String([]string{"--gogo-proto"}, "", "location of gogo import ")

	addDBAnnotation = goopt.Flag([]string{"--db"}, []string{}, "Add db annotations (tags)", "")

	verbose = goopt.Flag([]string{"-v", "--verbose"}, []string{}, "Enable verbose output", "")

	nameTest = goopt.String([]string{"--name_test"}, "", "perform name test using the --model_naming or --file_naming options")

	baseTemplates *packr.Box
	tableInfos    map[string]*dbmeta.ModelInfo
	au            aurora.Aurora
)

func init() {
	// Setup goopts
	goopt.Description = func() string {
		return "ORM and RESTful API generator for SQl databases"
	}
	goopt.Version = "v0.9.27 (08/04/2020)"
	goopt.Summary = `gen [-v] --sqltype=mysql --connstr "user:password@/dbname" --database <databaseName> --module=example.com/example [--json] [--gorm] [--guregu] [--generate-dao] [--generate-proj]
git fetch up
           sqltype - sql database type such as [ mysql, mssql, postgres, sqlite, etc. ]

`

	// Parse options
	goopt.Parse(nil)
}

func listTemplates() {
	for i, file := range baseTemplates.List() {
		fmt.Printf("   [%d] [%s]\n", i, file)
	}
}

func main() {
	//for i, arg := range os.Args {
	//	fmt.Printf("[%2d] %s\n", i, arg)
	//}
	au = aurora.NewAurora(false)
	dbmeta.InitColorOutput(au)

	baseTemplates = packr.New("gen", "./template")

	if *nameTest != "" {
		fmt.Printf("Running name test\n")
		fmt.Printf("table name: %s\n", *nameTest)

		fmt.Printf("modelNamingTemplate: %s\n", *modelNamingTemplate)
		result := dbmeta.Replace(*modelNamingTemplate, *nameTest)
		fmt.Printf("model: %s\n", result)

		fmt.Printf("fileNamingTemplate: %s\n", *fileNamingTemplate)
		result = dbmeta.Replace(*modelNamingTemplate, *nameTest)
		fmt.Printf("file: %s\n", result)

		fmt.Printf("fieldNamingTemplate: %s\n", *fieldNamingTemplate)
		result = dbmeta.Replace(*fieldNamingTemplate, *nameTest)
		fmt.Printf("field: %s\n", result)
		os.Exit(0)
		return
	}

	// Username is required
	if sqlConnStr == nil || *sqlConnStr == "" || *sqlConnStr == "nil" {
		fmt.Print(au.Red("sql connection string is required! Add it with --connstr=s\n\n"))
		fmt.Println(goopt.Usage())
		return
	}

	if sqlDatabase == nil || *sqlDatabase == "" || *sqlDatabase == "nil" {
		fmt.Print(au.Red("Database can not be null\n\n"))
		fmt.Println(goopt.Usage())
		return
	}

	db, err := initializeDB()
	if err != nil {
		fmt.Print(au.Red(fmt.Sprintf("Error in initializing db %v\n", err)))
		os.Exit(1)
		return
	}

	defer db.Close()

	var dbTables []string
	// parse or read tables
	if *sqlTable != "" {
		dbTables = strings.Split(*sqlTable, ",")
	} else {
		schemaTables, err := schema.TableNames(db)
		if err != nil {
			fmt.Print(au.Red(fmt.Sprintf("Error in fetching tables information from %s information schema from %s\n", *sqlType, *sqlConnStr)))
			os.Exit(1)
			return
		}
		for _, st := range schemaTables {
			dbTables = append(dbTables, st[1]) // s[0] == sqlDatabase
		}
	}

	if strings.HasPrefix(*modelNamingTemplate, "'") && strings.HasSuffix(*modelNamingTemplate, "'") {
		*modelNamingTemplate = strings.TrimSuffix(*modelNamingTemplate, "'")
		*modelNamingTemplate = strings.TrimPrefix(*modelNamingTemplate, "'")
	}

	var excludeDbTables []string

	if *excludeSQLTables != "" {
		excludeDbTables = strings.Split(*excludeSQLTables, ",")
	}

	conf := dbmeta.NewConfig(LoadTemplate)
	initialize(conf)

	err = loadDefaultDBMappings(conf)
	if err != nil {
		fmt.Print(au.Red(fmt.Sprintf("Error processing default mapping file error: %v\n", err)))
		os.Exit(1)
		return
	}

	tableInfos = dbmeta.LoadTableInfo(db, dbTables, excludeDbTables, conf)

	if len(tableInfos) == 0 {
		fmt.Print(au.Red(fmt.Sprintf("No tables loaded\n")))
		os.Exit(1)
	}

	fmt.Printf("Generating code for the following tables (%d)\n", len(tableInfos))
	i := 0
	for tableName := range tableInfos {
		fmt.Printf("[%d] %s\n", i, tableName)
		i++
	}

	conf.TableInfos = tableInfos
	conf.ContextMap["tableInfos"] = tableInfos

	if *verbose {
		listTemplates()
	}

	err = generate(conf)
	if err != nil {
		fmt.Print(au.Red(fmt.Sprintf("Error in executing generate %v\n", err)))
		os.Exit(1)
	}

	os.Exit(0)
}

func initializeDB() (db *sql.DB, err error) {
	db, err = sql.Open(*sqlType, *sqlConnStr)
	if err != nil {
		fmt.Print(au.Red(fmt.Sprintf("Error in open database: %v\n\n", err.Error())))
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		fmt.Print(au.Red(fmt.Sprintf("Error pinging database: %v\n\n", err.Error())))
		return
	}

	return
}

func initialize(conf *dbmeta.Config) {
	if outDir == nil || *outDir == "" {
		*outDir = "."
	}

	if modelPackageName == nil || *modelPackageName == "" {
		*modelPackageName = "model"
	}
	conf.SQLType = *sqlType
	conf.SQLDatabase = *sqlDatabase

	conf.AddJSONAnnotation = *addJSONAnnotation
	conf.AddGormAnnotation = *addGormAnnotation
	conf.AddProtobufAnnotation = *addProtobufAnnotation
	conf.AddDBAnnotation = *addDBAnnotation
	conf.JSONNameFormat = *jsonNameFormat
	conf.ProtobufNameFormat = *protoNameFormat
	conf.Verbose = *verbose
	conf.OutDir = *outDir
	conf.Overwrite = *overwrite

	conf.SQLConnStr = *sqlConnStr

	conf.ModelPackageName = *modelPackageName

	conf.FileNamingTemplate = *fileNamingTemplate
	conf.ModelNamingTemplate = *modelNamingTemplate
	conf.FieldNamingTemplate = *fieldNamingTemplate

	conf.Swagger.Title = fmt.Sprintf("Sample CRUD api for %s db", *sqlDatabase)
	conf.Swagger.Description = fmt.Sprintf("Sample CRUD api for %s db", *sqlDatabase)

	conf.JSONNameFormat = strings.ToLower(conf.JSONNameFormat)
	conf.XMLNameFormat = strings.ToLower(conf.XMLNameFormat)
	conf.ProtobufNameFormat = strings.ToLower(conf.ProtobufNameFormat)
}

func loadDefaultDBMappings(conf *dbmeta.Config) error {
	var err error
	var content []byte
	content, err = baseTemplates.Find("mapping.json")
	if err != nil {
		return err
	}

	err = dbmeta.ProcessMappings("internal", content, conf.Verbose)
	if err != nil {
		return err
	}
	return nil
}

func generate(conf *dbmeta.Config) error {
	var err error

	*jsonNameFormat = strings.ToLower(*jsonNameFormat)
	modelDir := filepath.Join(*outDir, *modelPackageName)

	err = os.MkdirAll(*outDir, 0777)
	if err != nil && !*overwrite {
		fmt.Print(au.Red(fmt.Sprintf("unable to create outDir: %s error: %v\n", *outDir, err)))
		return err
	}

	err = os.MkdirAll(modelDir, 0777)
	if err != nil && !*overwrite {
		fmt.Print(au.Red(fmt.Sprintf("unable to create modelDir: %s error: %v\n", modelDir, err)))
		return err
	}

	var ModelTmpl *dbmeta.GenTemplate
	if ModelTmpl, err = LoadTemplate("model.go.tmpl"); err != nil {
		fmt.Print(au.Red(fmt.Sprintf("Error loading template %v\n", err)))
		return err
	}

	*jsonNameFormat = strings.ToLower(*jsonNameFormat)

	// generate go files for each table
	for tableName, tableInfo := range tableInfos {

		if len(tableInfo.Fields) == 0 {
			if *verbose {
				fmt.Printf("[%d] Table: %s - No Fields Available\n", tableInfo.Index, tableName)
			}

			continue
		}

		modelInfo := conf.CreateContextForTableFile(tableInfo)

		modelFile := filepath.Join(modelDir, CreateGoSrcFileName(tableName))
		err = conf.WriteTemplate(ModelTmpl, modelInfo, modelFile)
		if err != nil {
			fmt.Print(au.Red(fmt.Sprintf("Error writing file: %v\n", err)))
			os.Exit(1)
		}
	}

	return nil
}

// CreateGoSrcFileName ensures name doesnt clash with go naming conventions like _test.go
func CreateGoSrcFileName(tableName string) string {
	name := dbmeta.Replace(*fileNamingTemplate, tableName)
	// name := inflection.Singular(tableName)

	if strings.HasSuffix(name, "_test") {
		name = name[0 : len(name)-5]
		name = name + "_tst"
	}
	return name + ".go"
}

// LoadTemplate return template from template dir, falling back to the embedded templates
func LoadTemplate(filename string) (tpl *dbmeta.GenTemplate, err error) {
	baseName := filepath.Base(filename)

	if *templateDir != "" {
		fpath := filepath.Join(*templateDir, filename)
		var b []byte
		b, err = ioutil.ReadFile(fpath)
		if err == nil {

			absPath, err := filepath.Abs(fpath)
			if err != nil {
				absPath = fpath
			}
			tpl = &dbmeta.GenTemplate{Name: "file://" + absPath, Content: string(b)}
			return tpl, nil
		}
	}

	content, err := baseTemplates.FindString(baseName)
	if err != nil {
		return nil, fmt.Errorf("%s not found internally", baseName)
	}
	if *verbose {
		fmt.Printf("Loaded template from app: %s\n", filename)
	}

	tpl = &dbmeta.GenTemplate{Name: "internal://" + filename, Content: content}
	return tpl, nil
}
