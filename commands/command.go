package commands

import (
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/beego/i18n"
	"github.com/howeyc/fsnotify"
	_ "github.com/lib/pq"
	"github.com/lifei6671/gocaptcha"
	"github.com/mindoc-org/mindoc/cache"
	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/models"
	"github.com/mindoc-org/mindoc/pkg/logger"
	"github.com/mindoc-org/mindoc/utils/filetil"
)

// RegisterDataBase 注册数据库
func RegisterDataBase() {
	logger.Info("正在初始化数据库配置.")
	dbadapter, _ := conf.GetString("db_adapter")

	var gormDB *gorm.DB
	var gormErr error

	if strings.EqualFold(dbadapter, "mysql") {
		host, _ := conf.GetString("db_host")
		database, _ := conf.GetString("db_database")
		username, _ := conf.GetString("db_username")
		password, _ := conf.GetString("db_password")

		timezone, _ := conf.GetString("timezone")
		_, err := time.LoadLocation(timezone)
		if err != nil {
			logger.Error("加载时区配置信息失败,请检查是否存在 ZONEINFO 环境变量->", err)
		}

		port, _ := conf.GetString("db_port")

		dataSource := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=%s", username, password, host, port, database, timezone)
		gormDB, gormErr = gorm.Open(mysql.Open(dataSource), models.GORMConfig())

	} else if strings.EqualFold(dbadapter, "sqlite3") {

		database, _ := conf.GetString("db_database")
		if strings.HasPrefix(database, "./") {
			database = filepath.Join(conf.WorkingDirectory, string(database[1:]))
		}
		if p, err := filepath.Abs(database); err == nil {
			database = p
		}

		dbPath := filepath.Dir(database)

		if _, err := os.Stat(dbPath); err != nil && os.IsNotExist(err) {
			_ = os.MkdirAll(dbPath, 0777)
		}

		gormDB, gormErr = gorm.Open(sqlite.Open(database), models.GORMConfig())

	} else if strings.EqualFold(dbadapter, "postgres") {
		host, _ := conf.GetString("db_host")
		database, _ := conf.GetString("db_database")
		username, _ := conf.GetString("db_username")
		password, _ := conf.GetString("db_password")
		sslmode, _ := conf.GetString("db_sslmode")

		timezone, _ := conf.GetString("timezone")
		_, err := time.LoadLocation(timezone)
		if err != nil {
			logger.Error("加载时区配置信息失败,请检查是否存在 ZONEINFO 环境变量->", err)
		}

		port, _ := conf.GetString("db_port")

		dataSource := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", username, password, host, port, database, sslmode)
		gormDB, gormErr = gorm.Open(postgres.Open(dataSource), models.GORMConfig())

	} else {
		logger.Error("不支持的数据库类型.")
		os.Exit(1)
	}

	if gormErr != nil {
		logger.Error("初始化GORM数据库失败->", gormErr)
		os.Exit(1)
	}
	models.SetDB(gormDB)

	// GORM 自动迁移
	if err := gormDB.AutoMigrate(
		&models.Member{}, &models.Book{}, &models.Relationship{},
		&models.Option{}, &models.Document{}, &models.Attachment{},
		&models.Logger{}, &models.MemberToken{}, &models.DocumentHistory{},
		&models.Migration{}, &models.Label{}, &models.Blog{},
		&models.Template{}, &models.Team{}, &models.TeamMember{},
		&models.TeamRelationship{}, &models.Itemsets{}, &models.Comment{},
			&models.CommentVote{}, &models.ContentReverseIndex{},
			&models.WorkWeixinAccount{}, &models.DingTalkAccount{},
	); err != nil {
		logger.Error("GORM 自动迁移失败 ->", err)
	}

 logger.Info("数据库初始化完成.")

	// 初始化默认选项
	if err := models.NewOption().Init(); err != nil {
		logger.Error("初始化选项失败 ->", err)
	}
}

// RegisterModel 注册Model
func RegisterModel() {
	gob.Register(models.Blog{})
	gob.Register(models.Document{})
	gob.Register(models.Template{})
	gob.Register(models.Member{})
}

// RegisterLogger 注册日志
func RegisterLogger(log string) {

	if log == "" {
		logPath, err := filepath.Abs(conf.GetDefaultString("log_path", conf.WorkingDir("runtime", "logs")))
		if err == nil {
			log = logPath
		} else {
			log = conf.WorkingDir("runtime", "logs")
		}
	}

	logPath := filepath.Join(log, "log.log")

	if _, err := os.Stat(log); os.IsNotExist(err) {
		_ = os.MkdirAll(log, 0755)
	}

	logger.Info("日志初始化完成. logPath=", logPath)
	_ = logPath
}

// RegisterCommand 注册orm命令行工具
func RegisterCommand() {

	if len(os.Args) >= 2 && os.Args[1] == "install" {
		ResolveCommand(os.Args[2:])
		Install()
	} else if len(os.Args) >= 2 && os.Args[1] == "version" {
		CheckUpdate()
		os.Exit(0)
	} else if len(os.Args) >= 2 && os.Args[1] == "update" {
		Update()
		os.Exit(0)
	} else if len(os.Args) >= 2 && os.Args[1] == "reindex" {
		ResolveCommand(os.Args[2:])
		fmt.Println("开始全量重建倒排索引...")
		if err := models.RebuildAllIndexes(); err != nil {
			fmt.Println("倒排索引重建失败，索引表可能处于部分重建状态，请重新执行 reindex:", err)
			os.Exit(1)
		}
		fmt.Println("倒排索引重建完成")
		os.Exit(0)
	}

}

func shouldInitializeMissingIndexes() bool {
	return !(len(os.Args) >= 2 && os.Args[1] == "reindex")
}

// RegisterFunction 注册模板函数和 i18n
func RegisterFunction() {
	i18nList, err := conf.GetString("i18n_list")
	if err != nil {
		logger.Error("error : failed to read i18n_list config ->", err)
		i18nList = ""
	}
	if i18nList == "" {
		logger.Error("error : config `i18n_list` is empty, please add config item like format: `i18n_list=zh-CN:简体中文|en-US:English`")
		i18nList = "zh-cn:简体中文|en-us:English|ru-ru:Русский"
	}

	langs := strings.Split(i18nList, "|")
	i18nMap := make(map[string]string)
	for _, langItem := range langs {
		langItemSplit := strings.Split(langItem, ":")
		if len(langItemSplit) < 2 {
			logger.Error("error: language config value `" + langItem + "` for `i18n_list` format error")
			continue
		}
		lang := langItemSplit[0]
		i18nMap[lang] = langItemSplit[1]
		if err := i18n.SetMessage(lang, conf.WorkingDir("conf", "lang", lang+".ini")); err != nil {
			logger.Error("Fail to set message file: " + err.Error())
			return
		}
	}

	i18nMapBytes, err := json.Marshal(i18nMap)
	if err != nil {
		logger.Error("error: Fail to marshal i18n map, " + err.Error())
		i18nMapBytes = []byte("{}")
	}

	_ = conf.SetConfigValue("i18n_map", string(i18nMapBytes))
}

// ResolveCommand 解析命令
func ResolveCommand(args []string) {
	flagSet := flag.NewFlagSet("MinDoc command: ", flag.ExitOnError)
	flagSet.StringVar(&conf.ConfigurationFile, "config", "", "MinDoc configuration file.")
	flagSet.StringVar(&conf.WorkingDirectory, "dir", "", "MinDoc working directory.")
	flagSet.StringVar(&conf.LogFile, "log", "", "MinDoc log file path.")

	if err := flagSet.Parse(args); err != nil {
		log.Fatal("解析命令失败 ->", err)
	}

	if conf.WorkingDirectory == "" {
		if p, err := filepath.Abs(os.Args[0]); err == nil {
			conf.WorkingDirectory = filepath.Dir(p)
		}
	}

	if conf.ConfigurationFile == "" {
		conf.ConfigurationFile = conf.WorkingDir("conf", "app.conf")
		config := conf.WorkingDir("conf", "app.conf.example")
		if !filetil.FileExists(conf.ConfigurationFile) && filetil.FileExists(config) {
			_ = filetil.CopyFile(conf.ConfigurationFile, config)
		}
	}
	if err := gocaptcha.ReadFonts(conf.WorkingDir("static", "fonts"), ".ttf"); err != nil {
		log.Fatal("读取字体文件时出错 -> ", err)
	}

	if err := conf.InitConfig(conf.ConfigurationFile); err != nil {
		log.Fatal("初始化配置失败:", err)
	}

	if conf.LogFile == "" {
		logPath, err := filepath.Abs(conf.GetDefaultString("log_path", conf.WorkingDir("runtime", "logs")))
		if err == nil {
			conf.LogFile = logPath
		} else {
			conf.LogFile = conf.WorkingDir("runtime", "logs")
		}
	}

	conf.AutoLoadDelay = conf.GetDefaultInt("config_auto_delay", 0)
	uploads := conf.WorkingDir("uploads")
	_ = os.MkdirAll(uploads, 0666)

	fonts := conf.WorkingDir("static", "fonts")
	if !filetil.FileExists(fonts) {
		log.Fatal("Font path not exist.")
	}
	if err := gocaptcha.ReadFonts(filepath.Join(conf.WorkingDirectory, "static", "fonts"), ".ttf"); err != nil {
		log.Fatal("读取字体失败 ->", err)
	}

	RegisterDataBase()
	RegisterCache()
	RegisterModel()
	RegisterLogger(conf.LogFile)
	models.InitExportPool()
	if shouldInitializeMissingIndexes() {
		models.InitializeMissingIndexes()
	}

	ModifyPassword()
}

// RegisterCache 注册缓存管道
func RegisterCache() {
	isOpenCache := conf.GetDefaultBool("cache", false)
	if !isOpenCache {
		cache.Init(&cache.NullCache{})
		return
	}
	logger.Info("正常初始化缓存配置.")
	cacheProvider, _ := conf.GetString("cache_provider")
	if cacheProvider == "file" {
		cacheFilePath := conf.GetDefaultString("cache_file_path", "./runtime/cache/")
		if strings.HasPrefix(cacheFilePath, "./") {
			cacheFilePath = filepath.Join(conf.WorkingDirectory, string(cacheFilePath[1:]))
		}
		fc := cache.NewFileCache()
		fc.CachePath = cacheFilePath
		fc.DirectoryLevel, _ = strconv.Atoi(conf.GetDefaultString("cache_file_dir_level", "2"))
		fc.EmbedExpiry, _ = strconv.Atoi(conf.GetDefaultString("cache_file_expiry", "120"))
		fc.FileSuffix = conf.GetDefaultString("cache_file_suffix", ".bin")

		_ = fc.StartAndGC("")
		cache.Init(fc)

	} else if cacheProvider == "memory" {
		cacheInterval := conf.GetDefaultInt("cache_memory_interval", 60)
		mc := cache.NewMemoryCache(time.Duration(cacheInterval)*time.Second, time.Duration(cacheInterval)*time.Second)
		cache.Init(mc)
	} else if cacheProvider == "redis" {
		conn := conf.GetDefaultString("cache_redis_host", "")
		pwd := conf.GetDefaultString("cache_redis_password", "")
		prefix := conf.GetDefaultString("cache_redis_prefix", "")
		dbNum := conf.GetDefaultInt("cache_redis_db", 0)

		rc, err := cache.NewRedisCache(conn, pwd, dbNum, prefix)
		if err != nil {
			logger.Error("初始化Redis缓存失败:", err)
			os.Exit(1)
		}
		cache.Init(rc)
	} else if cacheProvider == "memcache" {
		conn := conf.GetDefaultString("cache_memcache_host", "")
		mcc := cache.NewMemcacheCache(conn)
		cache.Init(mcc)
	} else {
		cache.Init(&cache.NullCache{})
		logger.Warn("不支持的缓存管道,缓存将禁用 ->", cacheProvider)
		return
	}
	logger.Info("缓存初始化完成.")
}

// RegisterAutoLoadConfig 自动加载配置文件
func RegisterAutoLoadConfig() {
	if conf.AutoLoadDelay > 0 {

		watcher, err := fsnotify.NewWatcher()

		if err != nil {
			logger.Error("创建配置文件监控器失败 ->", err)
		}
		go func() {
			for {
				select {
				case ev := <-watcher.Event:
					if ev.IsModify() {
						if err := conf.ReloadConfig(); err != nil {
							logger.Error("重新加载配置文件失败 ->", err)
							continue
						}
						RegisterCache()
						RegisterLogger("")
						logger.Info("配置文件已加载 ->", conf.ConfigurationFile)
					} else if ev.IsRename() {
						_ = watcher.WatchFlags(conf.ConfigurationFile, fsnotify.FSN_MODIFY|fsnotify.FSN_RENAME)
					}
					logger.Info(ev.String())
				case err := <-watcher.Error:
					logger.Error("配置文件监控器错误 ->", err)

				}
			}
		}()

		err = watcher.WatchFlags(conf.ConfigurationFile, fsnotify.FSN_MODIFY|fsnotify.FSN_RENAME)

		if err != nil {
			logger.Error("监控配置文件失败 ->", err)
		}
	}
}

// RegisterError 错误处理由 Gin 路由处理 (NoRoute/NoMethod)
func RegisterError() {
	// Gin 的 NoRoute 和 NoMethod 已在 SetupRouter 中注册
}

func init() {

	if configPath, err := filepath.Abs(conf.ConfigurationFile); err == nil {
		conf.ConfigurationFile = configPath
	}
	if err := gocaptcha.ReadFonts(conf.WorkingDir("static", "fonts"), ".ttf"); err != nil {
		log.Fatal("读取字体文件失败 ->", err)
	}
	gob.Register(models.Member{})

	if p, err := filepath.Abs(os.Args[0]); err == nil {
		conf.WorkingDirectory = filepath.Dir(p)
	}
}
