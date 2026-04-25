// Package models .
package models

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/mindoc-org/mindoc/conf"
	"github.com/mindoc-org/mindoc/pkg/logger"
	"github.com/mindoc-org/mindoc/utils"
	"github.com/beego/i18n"
	"gorm.io/gorm"
)

var LdapDefaultTimeout = 8 * time.Second

type Member struct {
	MemberId int    `gorm:"primaryKey;autoIncrement;uniqueIndex;column:member_id" json:"member_id"`
	Account  string `gorm:"size:100;uniqueIndex;column:account" json:"account"`
	RealName string `gorm:"size:255;column:real_name" json:"real_name"`
	Password string `gorm:"size:1000;column:password" json:"-"`
	//认证方式: local 本地数据库 /ldap LDAP
	AuthMethod  string `gorm:"column:auth_method;default:local;size:50" json:"auth_method"`
	Description string `gorm:"column:description;size:2000" json:"description"`
	Email       string `gorm:"size:100;column:email;uniqueIndex" json:"email"`
	Phone       string `gorm:"size:255;column:phone;default:null" json:"phone"`
	Avatar      string `gorm:"size:1000;column:avatar" json:"avatar"`
	//用户角色：0 超级管理员 /1 管理员/ 2 普通用户/ 3 只读用户 .
	Role          conf.SystemRole `gorm:"column:role;type:int;default:1;index" json:"role"`
	RoleName      string          `gorm:"-" json:"role_name"`
	Status        int             `gorm:"column:status;type:int;default:0" json:"status"` //用户状态：0 正常/1 禁用
	CreateTime    time.Time       `gorm:"type:datetime;column:create_time;autoCreateTime" json:"create_time"`
	CreateAt      int             `gorm:"type:int;column:create_at" json:"create_at"`
	LastLoginTime time.Time       `gorm:"type:datetime;column:last_login_time;default:null" json:"last_login_time"`
	//i18n
	Lang string `gorm:"-"`
}

// TableName 获取对应数据库表名.
func (m *Member) TableName() string {
	return conf.GetDatabasePrefix() + "members"
}

func NewMember() *Member {
	return &Member{}
}

// Login 用户登录.
func (m *Member) Login(account string, password string) (*Member, error) {
	member := &Member{}

	err := DB.Raw("select * from md_members where (account = ? or email = ?) and status = 0 limit 1;", account, account).Scan(member).Error

	if err != nil {
		if conf.GetDefaultBool("ldap_enable", false) {
			logger.Info("转入LDAP登陆 ->", account)
			return member.ldapLogin(account, password)
		} else if url, err := conf.GetString("http_login_url"); url != "" {
			logger.Info("转入 HTTP 接口登陆 ->", account)
			return member.httpLogin(account, password)
		} else {
			logger.Errorf("user login for `%s`: %s", account, err)
			return member, ErrMemberNoExist
		}
	}

	switch member.AuthMethod {
	case "local":
		ok, err := utils.PasswordVerify(member.Password, password)
		if ok && err == nil {
			m.ResolveRoleName()
			return member, nil
		}
	case "ldap":
		return member.ldapLogin(account, password)
	case "http":
		return member.httpLogin(account, password)
	default:
		return member, ErrMemberAuthMethodInvalid
	}

	return member, ErrorMemberPasswordError
}

// TmpLogin 用于钉钉临时登录
//func (m *Member) TmpLogin(account string) (*Member, error) {
//	o := orm.NewOrm()
//	member := &Member{}
//	err := o.Raw("select * from md_members where account = ? and status = 0 limit 1;", account).QueryRow(member)
//	if err != nil {
//		return member, ErrorMemberPasswordError
//	}
//	return member, nil
//}

// ldapLogin 通过LDAP登陆
func (m *Member) ldapLogin(account string, password string) (*Member, error) {
	if !conf.GetDefaultBool("ldap_enable", false) {
		return m, ErrMemberAuthMethodInvalid
	}
	var err error
	var ldapOpt ldap.DialOpt
	ldap_scheme := conf.GetDefaultString("ldap_scheme", "ldap")
	dialer := net.Dialer{Timeout: LdapDefaultTimeout}
	if ldap_scheme == "ldaps" {
		ldapOpt = ldap.DialWithTLSDialer(&tls.Config{InsecureSkipVerify: true}, &dialer)
	} else {
		ldapOpt = ldap.DialWithDialer(&dialer)
	}
	ldap_host, _ := conf.GetString("ldap_host")
	ldap_port := conf.GetDefaultInt("ldap_port", 3268)
	ldap_url := fmt.Sprintf("%s://%s:%d", ldap_scheme, ldap_host, ldap_port)
	lc, err := ldap.DialURL(ldap_url, ldapOpt)
	if err != nil {
		logger.Error("绑定 LDAP 用户失败 ->", err)
		return m, ErrLDAPConnect
	}
	defer lc.Close()
	ldapuser, _ := conf.GetString("ldap_user")
	ldappass, _ := conf.GetString("ldap_password")
	err = lc.Bind(ldapuser, ldappass)
	if err != nil {
		logger.Error("绑定 LDAP 用户失败 ->", err)
		return m, ErrLDAPFirstBind
	}
	ldapbase, _ := conf.GetString("ldap_base")
	ldapfilter, _ := conf.GetString("ldap_filter")
	ldapaccount, _ := conf.GetString("ldap_account")
	ldapmail, _ := conf.GetString("ldap_mail")
	// 判断account是否是email
	isEmail := false
	var email string
	ldapattr := ldapaccount
	if ok, err := regexp.MatchString(conf.RegexpEmail, account); ok && err == nil {
		isEmail = true
		email = account
		ldapattr = ldapmail
	}
	searchRequest := ldap.NewSearchRequest(
		ldapbase,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		// 修改objectClass通过配置文件获取值
		fmt.Sprintf("(&(%s)(%s=%s))", ldapfilter, ldapattr, account),
		[]string{"dn", "mail", "cn", "ou", "sAMAccountName"},
		nil,
	)
	searchResult, err := lc.Search(searchRequest)
	if err != nil {
		logger.Error("绑定 LDAP 用户失败 ->", err)
		return m, ErrLDAPSearch
	}
	if len(searchResult.Entries) != 1 {
		return m, ErrLDAPUserNotFoundOrTooMany
	}
	userdn := searchResult.Entries[0].DN
	err = lc.Bind(userdn, password)
	if err != nil {
		logger.Error("绑定 LDAP 用户失败 ->", err)
		return m, ErrorMemberPasswordError
	}

	ldap_cn := searchResult.Entries[0].GetAttributeValue("cn")
	ldap_mail := searchResult.Entries[0].GetAttributeValue(ldapmail)       // "mail"
	ldap_account := searchResult.Entries[0].GetAttributeValue(ldapaccount) // "sAMAccountName"

	m.RealName = ldap_cn
	m.Account = ldap_account
	m.AuthMethod = "ldap"
	// 如果ldap配置了email
	if len(ldap_mail) > 0 && strings.Contains(ldap_mail, "@") {
		// 如果member已配置email
		if len(m.Email) > 0 {
			// 如果member配置的email和ldap配置的email不同
			if m.Email != ldap_mail {
				return m, fmt.Errorf("ldap配置的email(%s)与数据库中已有email({%s})不同, 请联系管理员修改", ldap_mail, m.Email)
			}
		} else {
			// 如果member未配置email，则用ldap的email配置
			m.Email = ldap_mail
		}
	} else {
		// 如果ldap未配置email，则直接绑定到member
		if isEmail {
			m.Email = email
		}
	}
	if m.MemberId <= 0 {
		m.Avatar = "/static/images/headimgurl.jpg"
		m.Role = conf.SystemRole(conf.GetDefaultInt("ldap_user_role", 2))
		m.CreateTime = time.Now()

		err = m.Add()
		if err != nil {
			logger.Error("自动注册LDAP用户错误", err)
			return m, ErrorMemberPasswordError
		}
		m.ResolveRoleName()
	} else {
		// 更新ldap信息
		err = m.Update("account", "real_name", "email", "auth_method")
		if err != nil {
			logger.Error("LDAP更新用户信息失败", err)
			return m, errors.New("LDAP更新用户信息失败")
		}
		m.ResolveRoleName()
	}
	return m, nil
}

func (m *Member) httpLogin(account, password string) (*Member, error) {
	urlStr, _ := conf.GetString("http_login_url")
	if urlStr == "" {
		return nil, ErrMemberAuthMethodInvalid
	}

	val := url.Values{
		"account":  []string{account},
		"password": []string{password},
		"time":     []string{strconv.FormatInt(time.Now().Unix(), 10)},
	}
	h := md5.New()
	h.Write([]byte(val.Encode() + conf.GetDefaultString("http_login_secret", "")))

	val.Add("sn", hex.EncodeToString(h.Sum(nil)))

	resp, err := http.PostForm(urlStr, val)
	if err != nil {
		logger.Error("通过接口登录失败 -> ", urlStr, account, err)
		return nil, ErrHTTPServerFail
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取接口返回值失败 -> ", urlStr, account, err)
		return nil, ErrHTTPServerFail
	}
	logger.Info("HTTP 登录接口返回数据 ->", string(body))

	var result map[string]interface{}

	if err := json.Unmarshal(body, &result); err != nil {
		logger.Error("解析接口返回值失败 -> ", urlStr, account, string(body))
		return nil, ErrHTTPServerFail
	}

	if code, ok := result["errcode"]; !ok || code.(float64) != 200 {

		if msg, ok := result["message"]; ok {
			return nil, errors.New(msg.(string))
		}
		return nil, ErrHTTPServerFail
	}
	if m.MemberId <= 0 {
		member := NewMember()

		if email, ok := result["email"]; !ok || email == "" {
			return nil, errors.New("接口返回的数据缺少邮箱字段")
		} else {
			member.Email = email.(string)
		}

		if avatar, ok := result["avater"]; ok && avatar != "" {
			member.Avatar = avatar.(string)
		} else {
			member.Avatar = conf.URLForWithCdnImage("/static/images/headimgurl.jpg")
		}
		if realName, ok := result["real_name"]; ok && realName != "" {
			member.RealName = realName.(string)
		}
		member.Account = account
		member.Password = password
		member.AuthMethod = "http"
		member.Role = conf.SystemRole(conf.GetDefaultInt("ldap_user_role", 2))
		member.CreateTime = time.Now()
		if err := member.Add(); err != nil {
			logger.Error("自动注册用户错误", err)
			return m, ErrorMemberPasswordError
		}
		member.ResolveRoleName()
		*m = *member
	}
	return m, nil
}

// Add 添加一个用户.
func (m *Member) Add() error {
	if ok, err := regexp.MatchString(conf.RegexpAccount, m.Account); m.Account == "" || !ok || err != nil {
		return errors.New("账号只能由英文字母数字组成，且在3-50个字符")
	}
	if m.Email == "" {
		return errors.New("邮箱不能为空")
	}
	if ok, err := regexp.MatchString(conf.RegexpEmail, m.Email); !ok || err != nil || m.Email == "" {
		return errors.New("邮箱格式不正确")
	}
	if m.AuthMethod == "local" {
		if l := strings.Count(m.Password, ""); l < 6 || l >= 50 {
			return errors.New("密码不能为空且必须在6-50个字符之间")
		}
	}
	var c int64
	if err := DB.Table(m.TableName()).Where("email = ?", m.Email).Count(&c).Error; err == nil && c > 0 {
		return errors.New("邮箱已被使用")
	}

	hash, err := utils.PasswordHash(m.Password)

	if err != nil {
		logger.Error("加密用户密码失败 =>", err)
		return errors.New("加密用户密码失败")
	}

	m.Password = hash
	if m.AuthMethod == "" {
		m.AuthMethod = "local"
	}
	if err := DB.Create(m).Error; err != nil {
		logger.Error("保存用户数据到数据时失败 =>", err)
		return errors.New("保存用户失败")
	}
	m.ResolveRoleName()
	return nil
}

// Update 更新用户信息.
func (m *Member) Update(cols ...string) error {
	if m.Email == "" {
		return errors.New("邮箱不能为空")
	}
	var c int64
	if err := DB.Table(m.TableName()).Where("email = ? AND member_id != ?", m.Email, m.MemberId).Count(&c).Error; err == nil && c > 0 {
		return errors.New("邮箱已被使用")
	}
	if len(cols) > 0 {
		if err := DB.Model(m).Select(cols).Updates(m).Error; err != nil {
			logger.Error("保存用户信息失败=>", err)
			return errors.New("保存用户信息失败")
		}
	} else {
		if err := DB.Save(m).Error; err != nil {
			logger.Error("保存用户信息失败=>", err)
			return errors.New("保存用户信息失败")
		}
	}
	return nil
}

func (m *Member) Find(id int, cols ...string) (*Member, error) {
	query := DB.Table(m.TableName()).Where("member_id = ?", id)
	if len(cols) > 0 {
		query = query.Select(cols)
	}
	if err := query.First(m).Error; err != nil {
		return m, err
	}
	m.ResolveRoleName()
	return m, nil
}

func (m *Member) ResolveRoleName() {
	if m.Role == conf.MemberSuperRole {
		m.RoleName = i18n.Tr(m.Lang, "uc.super_admin")
	} else if m.Role == conf.MemberAdminRole {
		m.RoleName = i18n.Tr(m.Lang, "uc.admin")
	} else if m.Role == conf.MemberGeneralRole {
		m.RoleName = i18n.Tr(m.Lang, "uc.user")
	} else if m.Role == conf.MemberReaderRole {
		m.RoleName = i18n.Tr(m.Lang, "uc.read_usr")
	}
}

// 根据账号查找用户.
func (m *Member) FindByAccount(account string) (*Member, error) {
	err := DB.Table(m.TableName()).Where("account = ?", account).First(m).Error

	if err == nil {
		m.ResolveRoleName()
	}
	return m, err
}

// 批量查询用户
func (m *Member) FindByAccountList(accounts ...string) ([]*Member, error) {
	var members []*Member
	err := DB.Table(m.TableName()).Where("account IN ?", accounts).Find(&members).Error

	if err == nil {
		for _, item := range members {
			item.ResolveRoleName()
		}
	}
	return members, err
}

// 分页查找用户.
func (m *Member) FindToPager(pageIndex, pageSize int) ([]*Member, int, error) {
	var members []*Member

	offset := (pageIndex - 1) * pageSize

	var totalCount int64
	if err := DB.Table(m.TableName()).Count(&totalCount).Error; err != nil {
		return members, 0, err
	}

	if err := DB.Table(m.TableName()).Order("member_id DESC").Offset(offset).Limit(pageSize).Find(&members).Error; err != nil {
		return members, 0, err
	}

	for _, tm := range members {
		tm.Lang = m.Lang
		tm.ResolveRoleName()
	}
	return members, int(totalCount), nil
}

func (m *Member) IsAdministrator() bool {
	if m == nil || m.MemberId <= 0 {
		return false
	}
	return m.Role == 0 || m.Role == 1
}

// 根据指定字段查找用户.
func (m *Member) FindByFieldFirst(field string, value interface{}) (*Member, error) {
	err := DB.Table(m.TableName()).Where(field+" = ?", value).Order("member_id DESC").First(m).Error

	return m, err
}

// 校验用户.
func (m *Member) Valid(is_hash_password bool) error {

	//邮箱不能为空
	if m.Email == "" {
		return ErrMemberEmailEmpty
	}
	//用户描述必须小于500字
	if strings.Count(m.Description, "") > 500 {
		return ErrMemberDescriptionTooLong
	}
	if m.Role != conf.MemberGeneralRole && m.Role != conf.MemberSuperRole && m.Role != conf.MemberAdminRole && m.Role != conf.MemberReaderRole {
		return ErrMemberRoleError
	}
	if m.Status != 0 && m.Status != 1 {
		m.Status = 0
	}
	//邮箱格式校验
	if ok, err := regexp.MatchString(conf.RegexpEmail, m.Email); !ok || err != nil || m.Email == "" {
		return ErrMemberEmailFormatError
	}
	//如果是未加密密码，需要校验密码格式
	if !is_hash_password {
		if l := strings.Count(m.Password, ""); m.Password == "" || l > 50 || l < 6 {
			return ErrMemberPasswordFormatError
		}
	}
	//校验邮箱是否呗使用
	if member, err := NewMember().FindByFieldFirst("email", m.Account); err == nil && member.MemberId > 0 {
		if m.MemberId > 0 && m.MemberId != member.MemberId {
			return ErrMemberEmailExist
		}
		if m.MemberId <= 0 {
			return ErrMemberEmailExist
		}
	}

	if m.MemberId > 0 {
		//校验用户是否存在
		if _, err := NewMember().Find(m.MemberId); err != nil {
			return err
		}
	} else {
		//校验账号格式是否正确
		if ok, err := regexp.MatchString(conf.RegexpAccount, m.Account); m.Account == "" || !ok || err != nil {
			return ErrMemberAccountFormatError
		}
		//校验账号是否被使用
		if member, err := NewMember().FindByAccount(m.Account); err == nil && member.MemberId > 0 {
			return ErrMemberExist
		}
	}

	return nil
}

// 删除一个用户.
func (m *Member) Delete(oldId int, newId int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM md_dingtalk_accounts WHERE member_id = ?", oldId).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM md_workweixin_accounts WHERE member_id = ?", oldId).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM md_members WHERE member_id = ?", oldId).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE md_attachment SET create_at = ? WHERE create_at = ?", newId, oldId).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE md_books SET member_id = ? WHERE member_id = ?", newId, oldId).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE md_document_history SET member_id=? WHERE member_id = ?", newId, oldId).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE md_document_history SET modify_at=? WHERE modify_at = ?", newId, oldId).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE md_documents SET member_id = ? WHERE member_id = ?;", newId, oldId).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE md_documents SET modify_at = ? WHERE modify_at = ?", newId, oldId).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE md_blogs SET member_id = ? WHERE member_id = ?;", newId, oldId).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE md_blogs SET modify_at = ? WHERE modify_at = ?", newId, oldId).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE md_templates SET modify_at = ? WHERE modify_at = ?", newId, oldId).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE md_templates SET member_id = ? WHERE member_id = ?", newId, oldId).Error; err != nil {
			return err
		}
		if err := tx.Table("team_members").Where("member_id = ?", oldId).Delete(nil).Error; err != nil {
			return err
		}

		var relationshipList []*Relationship
		if err := tx.Table("relationships").Where("member_id = ?", oldId).Find(&relationshipList).Error; err != nil {
			return err
		}

		for _, relationship := range relationshipList {
			//如果存在创始人，则删除
			if relationship.RoleId == 0 {
				rel := NewRelationship()

				err := tx.Table("relationships").Where("book_id = ? AND member_id = ?", relationship.BookId, newId).First(rel).Error
				if err == nil {
					if err := tx.Delete(relationship).Error; err != nil {
						logger.Error(err)
					}
					relationship.RelationshipId = rel.RelationshipId
				}
				relationship.MemberId = newId
				relationship.RoleId = 0
				if err := tx.Save(relationship).Error; err != nil {
					logger.Error(err)
				}
			} else {
				if err := tx.Delete(relationship).Error; err != nil {
					logger.Error(err)
				}
			}
		}

		return nil
	})
}
