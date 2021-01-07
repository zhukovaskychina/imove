package sqlstore

import (
	"fmt"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/registry"
	"github.com/grafana/grafana/pkg/setting"
	"imove-server/server/log"
	"imove-server/server/sqlstore/migrations"
	"imove-server/server/sqlstore/migrator"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-xorm/xorm"

	_ "github.com/lib/pq"
)

var (
	x       *xorm.Engine
	dialect migrator.Dialect

	sqlog log.Logger = log.New("sqlstore")
)

const ContextSessionName = "db-session"

func init() {
	// This change will make xorm use an empty default schema for postgres and
	// by that mimic the functionality of how it was functioning before
	// xorm's changes above.
	xorm.DefaultPostgresSchema = ""

	registry.Register(&registry.Descriptor{
		Name:         "SqlStore",
		Instance:     &SqlStore{},
		InitPriority: registry.High,
	})
}

type SqlStore struct {
	Cfg *setting.Cfg `inject:""`
	Bus bus.Bus      `inject:""`

	dbCfg                       DatabaseConfig
	engine                      *xorm.Engine
	log                         log.Logger
	Dialect                     migrator.Dialect
	skipEnsureDefaultOrgAndUser bool
}

func (ss *SqlStore) Init() error {
	ss.log = log.New("sqlstore")
	ss.readConfig()

	engine, err := ss.getEngine()

	if err != nil {
		return fmt.Errorf("Fail to connect to database: %v", err)
	}

	ss.engine = engine
	ss.Dialect = migrator.NewDialect(ss.engine)

	// temporarily still set global var
	x = engine
	dialect = ss.Dialect
	migrator := migrator.NewMigrator(x)
	migrations.AddMigrations(migrator)

	if err := migrator.Start(); err != nil {
		return fmt.Errorf("Migration failed err: %v", err)
	}
	return nil
}

func (ss *SqlStore) buildExtraConnectionString(sep rune) string {
	if ss.dbCfg.UrlQueryParams == nil {
		return ""
	}

	var sb strings.Builder
	for key, values := range ss.dbCfg.UrlQueryParams {
		for _, value := range values {
			sb.WriteRune(sep)
			sb.WriteString(key)
			sb.WriteRune('=')
			sb.WriteString(value)
		}
	}
	return sb.String()
}

func (ss *SqlStore) buildConnectionString() (string, error) {
	cnnstr := ss.dbCfg.ConnectionString

	// special case used by integration tests
	if cnnstr != "" {
		return cnnstr, nil
	}

	switch ss.dbCfg.Type {

	case migrator.SQLITE:
		// special case for tests
		if !filepath.IsAbs(ss.dbCfg.Path) {
			ss.dbCfg.Path = filepath.Join(ss.Cfg.DataPath, ss.dbCfg.Path)
		}
		if err := os.MkdirAll(path.Dir(ss.dbCfg.Path), os.ModePerm); err != nil {
			return "", err
		}

		cnnstr = fmt.Sprintf("file:%s?cache=%s&mode=rwc", ss.dbCfg.Path, ss.dbCfg.CacheMode)
		cnnstr += ss.buildExtraConnectionString('&')
	default:
		return "", fmt.Errorf("Unknown database type: %s", ss.dbCfg.Type)
	}

	return cnnstr, nil
}

func (ss *SqlStore) getEngine() (*xorm.Engine, error) {
	connectionString, err := ss.buildConnectionString()

	if err != nil {
		return nil, err
	}

	sqlog.Info("Connecting to DB", "dbtype", ss.dbCfg.Type)
	engine, err := xorm.NewEngine(ss.dbCfg.Type, connectionString)
	if err != nil {
		return nil, err
	}

	engine.SetMaxOpenConns(ss.dbCfg.MaxOpenConn)
	engine.SetMaxIdleConns(ss.dbCfg.MaxIdleConn)
	engine.SetConnMaxLifetime(time.Second * time.Duration(ss.dbCfg.ConnMaxLifetime))

	// configure sql logging
	debugSql := ss.Cfg.Raw.Section("database").Key("log_queries").MustBool(false)
	if !debugSql {
		engine.SetLogger(&xorm.DiscardLogger{})
	} else {
		engine.SetLogger(NewXormLogger(log.LvlInfo, log.New("sqlstore.xorm")))
		engine.ShowSQL(true)
		engine.ShowExecTime(true)
	}

	return engine, nil
}

func (ss *SqlStore) readConfig() {
	sec := ss.Cfg.Raw.Section("database")

	cfgURL := sec.Key("url").String()
	if len(cfgURL) != 0 {
		dbURL, _ := url.Parse(cfgURL)
		ss.dbCfg.Type = dbURL.Scheme
		ss.dbCfg.Host = dbURL.Host

		pathSplit := strings.Split(dbURL.Path, "/")
		if len(pathSplit) > 1 {
			ss.dbCfg.Name = pathSplit[1]
		}

		userInfo := dbURL.User
		if userInfo != nil {
			ss.dbCfg.User = userInfo.Username()
			ss.dbCfg.Pwd, _ = userInfo.Password()
		}

		ss.dbCfg.UrlQueryParams = dbURL.Query()
	} else {
		ss.dbCfg.Type = sec.Key("type").String()
		ss.dbCfg.Host = sec.Key("host").String()
		ss.dbCfg.Name = sec.Key("name").String()
		ss.dbCfg.User = sec.Key("user").String()
		ss.dbCfg.ConnectionString = sec.Key("connection_string").String()
		ss.dbCfg.Pwd = sec.Key("password").String()
	}

	ss.dbCfg.MaxOpenConn = sec.Key("max_open_conn").MustInt(0)
	ss.dbCfg.MaxIdleConn = sec.Key("max_idle_conn").MustInt(2)
	ss.dbCfg.ConnMaxLifetime = sec.Key("conn_max_lifetime").MustInt(14400)

	ss.dbCfg.SslMode = sec.Key("ssl_mode").String()
	ss.dbCfg.CaCertPath = sec.Key("ca_cert_path").String()
	ss.dbCfg.ClientKeyPath = sec.Key("client_key_path").String()
	ss.dbCfg.ClientCertPath = sec.Key("client_cert_path").String()
	ss.dbCfg.ServerCertName = sec.Key("server_cert_name").String()
	ss.dbCfg.Path = sec.Key("path").MustString("data/MOVE.db")

	ss.dbCfg.CacheMode = sec.Key("cache_mode").MustString("private")
}

// Interface of arguments for testing db
type ITestDB interface {
	Helper()
	Fatalf(format string, args ...interface{})
}

type DatabaseConfig struct {
	Type             string
	Host             string
	Name             string
	User             string
	Pwd              string
	Path             string
	SslMode          string
	CaCertPath       string
	ClientKeyPath    string
	ClientCertPath   string
	ServerCertName   string
	ConnectionString string
	MaxOpenConn      int
	MaxIdleConn      int
	ConnMaxLifetime  int
	CacheMode        string
	UrlQueryParams   map[string][]string
}
